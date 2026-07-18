import { mkdir, mkdtemp, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { getDirectoryCompletions } from "../extensions/wt/directory-completion.ts";
import { expandPath } from "../extensions/wt/paths.ts";

describe("move path expansion", () => {
  const cwd = resolve("/workspace/repo/packages/app");
  const home = resolve("/home/example");

  it("expands the home directory", () => {
    expect(expandPath("~", cwd, home)).toBe(home);
    expect(expandPath("~/Develop/project", cwd, home)).toBe(join(home, "Develop", "project"));
  });

  it("preserves absolute paths", () => {
    const absolute = resolve("/other/repo");
    expect(expandPath(absolute, cwd, home)).toBe(absolute);
  });

  it("resolves relative paths from the session cwd", () => {
    expect(expandPath("../other", cwd, home)).toBe(resolve(cwd, "../other"));
  });
});

describe("move directory completion", () => {
  it("lists only matching directories for relative, home, and absolute prefixes", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-mv-completion-"));
    const cwd = join(root, "workspace", "current");
    const home = join(root, "home");
    const absolute = join(root, "absolute-target");
    await mkdir(cwd, { recursive: true });
    await mkdir(join(root, "workspace", "relative-target"), { recursive: true });
    await mkdir(join(cwd, "foo bar"));
    await mkdir(join(home, "Develop"), { recursive: true });
    await mkdir(absolute);
    await writeFile(join(home, "Development.txt"), "not a directory");

    expect(await getDirectoryCompletions("../rel", cwd, home)).toEqual([
      { value: "../relative-target/", label: "relative-target/" },
    ]);
    expect(await getDirectoryCompletions("~/Dev", cwd, home)).toEqual([
      { value: "~/Develop/", label: "Develop/" },
    ]);
    expect(await getDirectoryCompletions(join(root, "abs"), cwd, home)).toEqual([
      { value: `${absolute}/`, label: "absolute-target/" },
    ]);
    expect(await getDirectoryCompletions('foo" b', cwd, home)).toEqual([
      { value: '"foo bar/"', label: "foo bar/" },
    ]);
  });

  it("includes symlinks to directories", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-mv-completion-"));
    const target = join(root, "target");
    await mkdir(target);
    await symlink(target, join(root, "linked"));

    expect(await getDirectoryCompletions("lin", root, root)).toEqual([
      { value: "linked/", label: "linked/" },
    ]);
  });
});
