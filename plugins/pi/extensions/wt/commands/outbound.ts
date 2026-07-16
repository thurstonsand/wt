import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { isPersisted, migrateSession } from "../migration.ts";
import { notifyStderr } from "../notifications.ts";
import { runWt } from "../process.ts";

export async function runOutbound(
  command: "fork" | "checkout" | "co",
  args: string[],
  ctx: ExtensionCommandContext,
): Promise<void> {
  const result = await runWt([command, ...args], ctx.cwd);
  notifyStderr(ctx, result);
  if (!result.cd) throw new Error(`wt ${command} did not report a destination`);

  if (!isPersisted(ctx)) {
    ctx.ui.notify(
      `Worktree ready at ${result.cd}. This in-memory session cannot move; use \`wt cd\` in a terminal.`,
      "warning",
    );
    return;
  }
  await migrateSession(ctx, result.cd);
}
