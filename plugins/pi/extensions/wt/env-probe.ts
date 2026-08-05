import { spawn } from "node:child_process";
import { basename } from "node:path";
import type { Readable } from "node:stream";
import {
  applyEnvDelta,
  computeEnvDelta,
  type EnvDelta,
  PROBE_SENTINEL,
  parseProbeOutput,
} from "./env-delta.ts";

const PROBE_TIMEOUT_MS = 10_000;

function singleQuote(value: string): string {
  return `'${value.replaceAll("'", `'\\''`)}'`;
}

// One shell, two captures: a fresh shell mints per-invocation values (session
// ids, prompt state) that would otherwise read as differences. Capturing both
// sides inside the same shell cancels that noise out. The shell must be
// interactive (`-i`): direnv/mise/zoxide only install their hooks from the
// interactive rc files.
function hookInvocation(shell: string): string | undefined {
  if (shell === "zsh") {
    return "for f in $chpwd_functions $precmd_functions; do (( $+functions[$f] )) && $f; done";
  }
  // bash has no chpwd/precmd arrays; direnv and mise hook PROMPT_COMMAND, which
  // is a string before bash 5.1 and an array after. `"${PROMPT_COMMAND[@]}"`
  // reads both shapes, since a scalar expands as its own element zero.
  // biome-ignore lint/suspicious/noTemplateCurlyInString: shell expansion, not a JS template
  if (shell === "bash") return 'for c in "${PROMPT_COMMAND[@]}"; do eval "$c"; done';
  return undefined;
}

export function probeScript(shellPath: string, destination: string): string | undefined {
  const hooks = hookInvocation(basename(shellPath));
  if (!hooks) return undefined;
  return [
    "env -0",
    `printf '%s\\0' ${singleQuote(PROBE_SENTINEL)}`,
    `cd -- ${singleQuote(destination)} || exit 1`,
    `{ ${hooks}; } >/dev/null 2>&1`,
    "env -0",
  ].join("\n");
}

function runProbe(shellPath: string, script: string, cwd: string): Promise<string> {
  return new Promise((resolve, reject) => {
    // stderr is discarded: interactive shells complain about a missing tty
    // ("can't change option: zle") and hooks chatter on every directory change.
    const child = spawn(shellPath, ["-ilc", script], {
      cwd,
      detached: true,
      shell: false,
      stdio: ["ignore", "pipe", "ignore"],
    });
    const timer = setTimeout(() => {
      timedOut = true;
      if (child.pid !== undefined) process.kill(-child.pid, "SIGKILL");
    }, PROBE_TIMEOUT_MS);

    let stdout = "";
    let timedOut = false;
    let spawnError: Error | undefined;
    const stdoutStream = child.stdio[1] as Readable;
    stdoutStream.setEncoding("utf8");
    stdoutStream.on("data", (chunk: string) => {
      stdout += chunk;
    });
    child.on("error", (error) => {
      spawnError = error;
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      if (timedOut)
        reject(
          new Error(`${basename(shellPath)} did not answer within ${PROBE_TIMEOUT_MS / 1000}s`),
        );
      else if (spawnError) reject(spawnError);
      else if (code !== 0) reject(new Error(`${basename(shellPath)} exited with code ${code}`));
      else resolve(stdout);
    });
  });
}

// Fires the destination directory's shell hooks (direnv, mise, and whatever
// else the user wires into the standard entry points) and folds the resulting
// environment into this process. pi's bash tool rebuilds its environment from
// `process.env` on every invocation, so this reaches every later command.
export async function reconcileDirectoryEnv(
  oldCwd: string,
  newCwd: string,
  shellPath = process.env.SHELL,
): Promise<EnvDelta | undefined> {
  if (!shellPath) return undefined;
  const script = probeScript(shellPath, newCwd);
  if (!script) return undefined;

  const delta = computeEnvDelta(parseProbeOutput(await runProbe(shellPath, script, oldCwd)));
  applyEnvDelta(delta, process.env);
  return delta;
}
