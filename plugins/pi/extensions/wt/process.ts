import { spawn } from "node:child_process";
import { homedir } from "node:os";
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

export function execWt(args: string[], cwd: string): Promise<WtResult> {
  return new Promise((resolve) => {
    const child = spawn("wt", args, {
      cwd,
      shell: false,
      stdio: ["ignore", "pipe", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    let cd = "";
    let spawnError: Error | undefined;
    const stdoutStream = child.stdio[1] as Readable;
    const stderrStream = child.stdio[2] as Readable;
    const cdStream = child.stdio[3] as Readable;

    stdoutStream.setEncoding("utf8");
    stderrStream.setEncoding("utf8");
    cdStream.setEncoding("utf8");
    stdoutStream.on("data", (chunk: string) => {
      stdout += chunk;
    });
    stderrStream.on("data", (chunk: string) => {
      stderr += chunk;
    });
    cdStream.on("data", (chunk: string) => {
      cd += chunk;
    });
    child.on("error", (error) => {
      spawnError = error;
    });
    child.on("close", (code) => {
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
      resolve({ stdout, stderr, ...(cd.trim() ? { cd: cd.trim() } : {}), code: code ?? 1 });
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
