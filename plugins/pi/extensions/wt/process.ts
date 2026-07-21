import { spawn } from "node:child_process";
import { closeSync, mkdtempSync, openSync, readFileSync, rmSync } from "node:fs";
import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import type { Readable } from "node:stream";

export interface WtResult {
  stdout: string;
  stderr: string;
  cd?: string;
  code: number;
  spawnErrorCode?: string;
}

export class WtError extends Error {
  readonly result: WtResult;

  constructor(result: WtResult) {
    const message = result.stderr.trim() || `wt exited with code ${result.code}`;
    super(message);
    this.name = "WtError";
    this.result = result;
  }
}

// `wt` reports its cd target on fd 3 (see internal/shell/wrapper.go), so we run
// it with a fourth stdio entry to capture that channel.
//
// That fourth entry is backed by a temp file rather than the obvious "pipe".
// The reason is a Bun bug: for stdio beyond stderr, Bun does not hand the child
// a plain OS pipe the way Node does. It creates a unix-domain socket and
// connects to it *asynchronously* from the parent. That connect intermittently
// fails with ENOENT ("Failed to connect"), and the failure lands on Bun's own
// internal socket object, so it surfaces as a detached unhandledRejection that
// no try/catch here can intercept. pi installs no unhandledRejection handler,
// so the whole process dies. Fast typing in a `/wt` slash command spawns one
// completion per keystroke, which turns the ~1-in-20 flake into a near-certain
// crash; a single command can trip it too, just rarely. Node is unaffected.
//
// Handing Bun an already-open fd gives it nothing to connect: it just dups the
// descriptor into the child, so the buggy socket path is never entered. `wt`'s
// fd-3 contract is unchanged; only what fd 3 points at differs.
//
// Remove this once Bun spawns extra stdio pipes reliably: drop the temp file
// and restore `stdio: ["ignore", "pipe", "pipe", "pipe"]` reading fd 3 directly.
// Regression repro at scripts/repro-bun-fd3-crash.mjs.
export function execWt(args: string[], cwd: string): Promise<WtResult> {
  return new Promise((resolve) => {
    const cdDir = mkdtempSync(join(tmpdir(), "pi-wt-cd-"));
    const cdPath = join(cdDir, "cd");
    const cdFd = openSync(cdPath, "w");

    const collectCd = (): string => {
      try {
        closeSync(cdFd);
      } catch {}
      let cd = "";
      try {
        cd = readFileSync(cdPath, "utf8");
      } catch {}
      try {
        rmSync(cdDir, { recursive: true, force: true });
      } catch {}
      return cd;
    };

    const child = spawn("wt", args, {
      cwd,
      shell: false,
      stdio: ["ignore", "pipe", "pipe", cdFd],
    });
    let stdout = "";
    let stderr = "";
    let spawnError: Error | undefined;
    const stdoutStream = child.stdio[1] as Readable;
    const stderrStream = child.stdio[2] as Readable;

    stdoutStream.setEncoding("utf8");
    stderrStream.setEncoding("utf8");
    stdoutStream.on("data", (chunk: string) => {
      stdout += chunk;
    });
    stderrStream.on("data", (chunk: string) => {
      stderr += chunk;
    });
    child.on("error", (error) => {
      spawnError = error;
    });
    child.on("close", (code) => {
      const cd = collectCd();
      if (spawnError) {
        const error = spawnError as NodeJS.ErrnoException;
        resolve({
          stdout,
          stderr:
            stderr || (error.code === "ENOENT" ? "wt binary not found on PATH" : error.message),
          code: 127,
          ...(error.code ? { spawnErrorCode: error.code } : {}),
        });
        return;
      }
      resolve({
        stdout,
        stderr,
        ...(cd.trim() ? { cd: cd.trim() } : {}),
        code: code ?? 1,
      });
    });
  });
}

export async function isWtInstalled(): Promise<boolean> {
  return (await execWt(["--version"], homedir())).spawnErrorCode === undefined;
}

export async function runWt(args: string[], cwd: string): Promise<WtResult> {
  const result = await execWt(args, cwd);
  if (result.code !== 0) throw new WtError(result);
  return result;
}
