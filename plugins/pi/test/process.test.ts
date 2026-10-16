import { existsSync } from "node:fs";
import { chmod, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { execWt, isWtInstalled } from "../extensions/wt/process.ts";

const originalPath = process.env.PATH;
const originalPlatform = process.platform;

afterEach(() => {
  process.env.PATH = originalPath;
  Object.defineProperty(process, "platform", { value: originalPlatform });
});

async function fakeBinary(dir: string, name: string, script: string): Promise<void> {
  const path = join(dir, name);
  await writeFile(path, `#!/bin/sh\n${script}\n`);
  await chmod(path, 0o755);
}

describe("wt process", () => {
  it("detects an installed wt binary", async () => {
    const bin = await mkdtemp(join(tmpdir(), "pi-wt-bin-"));
    const wt = join(bin, "wt");
    await writeFile(wt, "#!/bin/sh\nexit 0\n");
    await chmod(wt, 0o755);
    process.env.PATH = bin;

    await expect(isWtInstalled()).resolves.toBe(true);
  });

  it("detects when wt is absent from PATH", async () => {
    process.env.PATH = await mkdtemp(join(tmpdir(), "pi-wt-bin-"));
    await expect(isWtInstalled()).resolves.toBe(false);
  });

  it("captures the shell cd directive from fd 3", async () => {
    const bin = await mkdtemp(join(tmpdir(), "pi-wt-bin-"));
    const wt = join(bin, "wt");
    await writeFile(
      wt,
      '#!/bin/sh\nprintf "stdout\\n"\nprintf "stderr\\n" >&2\nprintf "/destination/worktree" >&3\nexit 7\n',
    );
    await chmod(wt, 0o755);
    process.env.PATH = `${bin}${delimiter}${originalPath ?? ""}`;

    await expect(execWt(["anything"], bin)).resolves.toEqual({
      stdout: "stdout\n",
      stderr: "stderr\n",
      cd: "/destination/worktree",
      code: 7,
    });
  });

  describe("on Windows", () => {
    // Stands in for Windows Terminal's `wt.exe` alias: any call leaves a marker.
    async function windowsPath(withGitWt: boolean): Promise<{ bin: string; touched: string }> {
      Object.defineProperty(process, "platform", { value: "win32" });
      const bin = await mkdtemp(join(tmpdir(), "pi-wt-bin-"));
      const touched = join(bin, "windows-terminal-ran");
      await fakeBinary(bin, "wt", `: > "${touched}"`);
      if (withGitWt) await fakeBinary(bin, "git-wt", 'printf "git-wt %s" "$1"');
      process.env.PATH = bin;
      return { bin, touched };
    }

    it("runs git-wt and never touches the wt on PATH", async () => {
      const { bin, touched } = await windowsPath(true);

      await expect(isWtInstalled()).resolves.toBe(true);
      await expect(execWt(["--version"], bin)).resolves.toMatchObject({
        stdout: "git-wt --version",
        code: 0,
      });
      expect(existsSync(touched)).toBe(false);
    });

    it("reports git-wt missing even when a wt is on PATH", async () => {
      const { bin, touched } = await windowsPath(false);

      await expect(isWtInstalled()).resolves.toBe(false);
      await expect(execWt(["--version"], bin)).resolves.toMatchObject({
        stderr: "git-wt binary not found on PATH",
        code: 127,
      });
      expect(existsSync(touched)).toBe(false);
    });
  });
});
