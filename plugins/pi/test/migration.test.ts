import { mkdir, mkdtemp, readFile, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, join } from "node:path";
import { SessionManager } from "@earendil-works/pi-coding-agent";
import { afterEach, describe, expect, it } from "vitest";
import { pathContains } from "../extensions/wt/paths.ts";
import {
  discardPreparedMigration,
  finishMigration,
  serializeSession,
  writeMigration,
} from "../extensions/wt/session-file.ts";

const originalAgentDir = process.env.PI_CODING_AGENT_DIR;

afterEach(() => {
  if (originalAgentDir === undefined) delete process.env.PI_CODING_AGENT_DIR;
  else process.env.PI_CODING_AGENT_DIR = originalAgentDir;
});

async function fixtureSession(root: string): Promise<{ manager: SessionManager; path: string }> {
  const sourceCwd = join(root, "source");
  const sourceStore = join(root, "sessions-source");
  await mkdir(sourceCwd, { recursive: true });
  await mkdir(sourceStore, { recursive: true });
  const path = join(sourceStore, "2026-07-15T00-00-00-000Z_session-id.jsonl");
  const header = {
    type: "session",
    version: 3,
    id: "session-id",
    timestamp: "2026-07-15T00:00:00.000Z",
    cwd: sourceCwd,
    parentSession: "/original/parent.jsonl",
    futureHeader: { preserved: true },
  };
  const entry = {
    type: "custom",
    id: "deadbeef",
    parentId: null,
    timestamp: "2026-07-15T00:00:01.000Z",
    customType: "fixture",
    data: { intact: true },
    futureEntry: { preserved: true },
  };
  await writeFile(path, `${JSON.stringify(header)}\n${JSON.stringify(entry)}\n`);
  return { manager: SessionManager.open(path), path };
}

function sessionSnapshot(manager: SessionManager) {
  const header = manager.getHeader();
  if (!header) throw new Error("fixture session has no header");
  return { header, entries: manager.getEntries() };
}

function prepareFixtureMigration(manager: SessionManager, oldPath: string, destination: string) {
  return writeMigration(sessionSnapshot(manager), oldPath, manager.getCwd(), destination);
}

describe("session migration", () => {
  it("rewrites only cwd while preserving identity, filename, and entries", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-"));
    process.env.PI_CODING_AGENT_DIR = join(root, "agent");
    const { manager, path: oldPath } = await fixtureSession(root);
    const destination = join(root, "destination");
    await mkdir(destination, { recursive: true });

    const migration = await prepareFixtureMigration(manager, oldPath, destination);
    const lines = (await readFile(migration.newPath, "utf8"))
      .trim()
      .split("\n")
      .map((line) => JSON.parse(line) as Record<string, unknown>);

    expect(basename(migration.newPath)).toBe(basename(oldPath));
    expect(lines[0]).toMatchObject({
      id: "session-id",
      timestamp: "2026-07-15T00:00:00.000Z",
      cwd: migration.newCwd,
      parentSession: "/original/parent.jsonl",
      futureHeader: { preserved: true },
    });
    expect(lines[1]).toMatchObject({
      id: "deadbeef",
      data: { intact: true },
      futureEntry: { preserved: true },
    });

    await finishMigration(migration);
    await expect(readFile(oldPath, "utf8")).rejects.toThrow();
    expect(await readFile(migration.newPath, "utf8")).toContain('"id":"session-id"');
  });

  it("accepts a source session that pi has not flushed yet", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-"));
    process.env.PI_CODING_AGENT_DIR = join(root, "agent");
    const { manager, path: oldPath } = await fixtureSession(root);
    const destination = join(root, "destination");
    await mkdir(destination, { recursive: true });

    const migration = await prepareFixtureMigration(manager, oldPath, destination);
    await unlink(oldPath);

    await expect(finishMigration(migration)).resolves.toBeUndefined();
    expect(await readFile(migration.newPath, "utf8")).toContain('"id":"session-id"');
  });

  it("discards a prepared copy when switching is cancelled", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-"));
    process.env.PI_CODING_AGENT_DIR = join(root, "agent");
    const { manager, path: oldPath } = await fixtureSession(root);
    const destination = join(root, "destination");
    await mkdir(destination, { recursive: true });

    const migration = await prepareFixtureMigration(manager, oldPath, destination);
    await discardPreparedMigration(migration);

    await expect(readFile(migration.newPath, "utf8")).rejects.toThrow();
    expect(await readFile(oldPath, "utf8")).toContain('"id":"session-id"');
  });

  it("serializes a snapshot without changing lineage", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-"));
    const { manager } = await fixtureSession(root);
    const serialized = serializeSession(sessionSnapshot(manager), "/new/cwd");
    expect(serialized).toContain('"cwd":"/new/cwd"');
    expect(serialized).toContain('"parentSession":"/original/parent.jsonl"');
  });

  it("recognizes sessions launched below a worktree root", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-"));
    const worktree = join(root, "worktree");
    const nested = join(worktree, "src", "package");
    await mkdir(nested, { recursive: true });
    expect(await pathContains(worktree, nested)).toBe(true);
    expect(await pathContains(nested, worktree)).toBe(false);
  });
});
