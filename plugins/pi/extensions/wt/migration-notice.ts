import { basename } from "node:path";
import type { PreparedMigration } from "./session-file.ts";

export const MIGRATION_NOTICE_TYPE = "wt-session-migration";

export interface MigrationNoticeDetails {
  destinationName: string;
  deletedWorktree?: string;
}

export function migrationNotice(migration: PreparedMigration, deletedWorktree?: string) {
  let content = `Session migrated to \`${migration.newCwd}\`; paths under \`${migration.oldCwd}\` now resolve under \`${migration.newCwd}\`.`;
  if (deletedWorktree) {
    content += ` The old worktree directory \`${deletedWorktree}\` has been deleted.`;
  }
  return {
    customType: MIGRATION_NOTICE_TYPE,
    content,
    display: true,
    details: {
      destinationName: basename(migration.newCwd),
      ...(deletedWorktree ? { deletedWorktree } : {}),
    } satisfies MigrationNoticeDetails,
  };
}
