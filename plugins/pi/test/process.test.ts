import { chmod, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { execWt } from "../extensions/wt/process.ts";

const originalPath = process.env.PATH;

afterEach(() => {
  process.env.PATH = originalPath;
});

describe("wt process", () => {
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
});
