import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { reconcileDirectoryEnv } from "./env-probe.ts";
import { migrationNotice } from "./migration-notice.ts";
import { errorText, notifyStderr } from "./notifications.ts";
import type { WtResult } from "./process.ts";
import {
  discardPreparedMigration,
  finishMigration,
  type PreparedMigration,
  prepareMigration,
} from "./session-file.ts";

export interface MigrationOptions {
  afterSwitch?: () => Promise<WtResult>;
  deletedWorktree?: string;
  warning?: string;
}

export function isPersisted(ctx: ExtensionCommandContext): boolean {
  return ctx.sessionManager.getSessionFile() !== undefined;
}

export async function migrateSession(
  ctx: ExtensionCommandContext,
  destination: string,
  options: MigrationOptions = {},
): Promise<void> {
  let migration: PreparedMigration;
  try {
    migration = await prepareMigration(ctx, destination);
  } catch (error) {
    ctx.ui.notify(errorText(error), "error");
    return;
  }

  let result: { cancelled: boolean };
  try {
    result = await ctx.switchSession(migration.newPath, {
      withSession: async (replacementCtx) => {
        try {
          await finishMigration(migration);
        } catch (error) {
          replacementCtx.ui.notify(
            `Session moved, but the old file could not be removed: ${errorText(error)}`,
            "error",
          );
        }

        try {
          await reconcileDirectoryEnv(migration.oldCwd, migration.newCwd);
        } catch (error) {
          replacementCtx.ui.notify(
            `Session moved, but ${migration.newCwd}'s shell environment could not be loaded: ${errorText(error)}`,
            "warning",
          );
        }

        let deleted = false;
        if (options.afterSwitch) {
          try {
            notifyStderr(replacementCtx, await options.afterSwitch());
            deleted = true;
          } catch (error) {
            replacementCtx.ui.notify(errorText(error), "error");
          }
        }

        try {
          await replacementCtx.sendMessage(
            migrationNotice(migration, deleted ? options.deletedWorktree : undefined),
          );
        } catch (error) {
          replacementCtx.ui.notify(
            `Session moved, but its migration notice failed: ${errorText(error)}`,
            "error",
          );
        }
        if (options.warning) replacementCtx.ui.notify(options.warning, "warning");
      },
    });
  } catch (error) {
    await discardPreparedMigration(migration);
    ctx.ui.notify(errorText(error), "error");
    return;
  }

  if (!result.cancelled) return;
  await discardPreparedMigration(migration);
  ctx.ui.notify("Session migration cancelled", "warning");
}
