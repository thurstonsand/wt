import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { isPersisted, migrateSession } from "./migration.ts";
import { notifyResult } from "./notifications.ts";
import { pathContains } from "./paths.ts";
import type { WtResult } from "./process.ts";

export interface SourceRemovalOptions {
  sourcePath: string;
  destination: string | undefined;
  finish(): Promise<WtResult>;
  inMemoryWarning: string;
}

export async function removeSourceAfterSessionHandoff(
  ctx: ExtensionCommandContext,
  options: SourceRemovalOptions,
): Promise<void> {
  const sessionInSource = await pathContains(options.sourcePath, ctx.cwd);
  if (!isPersisted(ctx) || !sessionInSource) {
    notifyResult(ctx, await options.finish());
    if (!isPersisted(ctx) && sessionInSource) {
      ctx.ui.notify(options.inMemoryWarning, "warning");
    }
    return;
  }

  if (!options.destination) throw new Error("wt did not report a landing worktree");
  await migrateSession(ctx, options.destination, {
    deletedWorktree: options.sourcePath,
    afterSwitch: options.finish,
  });
}
