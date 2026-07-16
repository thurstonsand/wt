import { mkdir, unlink, writeFile } from "node:fs/promises";
import { basename, dirname, resolve } from "node:path";
import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import {
  type SessionEntry,
  type SessionHeader,
  SessionManager,
} from "@earendil-works/pi-coding-agent";
import { canonicalPath } from "./paths.ts";

export interface SessionSnapshot {
  header: SessionHeader;
  entries: SessionEntry[];
}

export interface PreparedMigration {
  oldCwd: string;
  oldPath: string;
  newCwd: string;
  newPath: string;
}

function snapshotCurrentSession(ctx: ExtensionCommandContext): SessionSnapshot {
  const header = ctx.sessionManager.getHeader();
  if (!header) throw new Error("The current session has no valid header");
  return { header, entries: ctx.sessionManager.getEntries() };
}

export function serializeSession(snapshot: SessionSnapshot, cwd: string): string {
  const header: SessionHeader = { ...snapshot.header, cwd };
  return `${[header, ...snapshot.entries].map((entry) => JSON.stringify(entry)).join("\n")}\n`;
}

export async function writeMigration(
  snapshot: SessionSnapshot,
  oldPath: string,
  oldCwd: string,
  destination: string,
): Promise<PreparedMigration> {
  const newCwd = await canonicalPath(destination);
  const destinationStore = SessionManager.create(newCwd).getSessionDir();
  const newPath = resolve(destinationStore, basename(oldPath));
  if (resolve(oldPath) === newPath) {
    throw new Error(`The session is already stored for ${newCwd}`);
  }

  await mkdir(dirname(newPath), { recursive: true });
  await writeFile(newPath, serializeSession(snapshot, newCwd), {
    encoding: "utf8",
    flag: "w",
    mode: 0o600,
  });

  return { oldCwd, oldPath, newCwd, newPath };
}

export async function prepareMigration(
  ctx: ExtensionCommandContext,
  destination: string,
): Promise<PreparedMigration> {
  const oldPath = ctx.sessionManager.getSessionFile();
  if (!oldPath) throw new Error("The current pi session is not persisted");
  return writeMigration(
    snapshotCurrentSession(ctx),
    oldPath,
    ctx.sessionManager.getCwd(),
    destination,
  );
}

export async function discardPreparedMigration(migration: PreparedMigration): Promise<void> {
  await unlink(migration.newPath).catch(() => undefined);
}

export async function finishMigration(migration: PreparedMigration): Promise<void> {
  try {
    await unlink(migration.oldPath);
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") throw error;
  }
}
