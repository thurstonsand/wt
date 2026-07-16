import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { injectFlag } from "../args.ts";
import { isPersisted, migrateSession } from "../migration.ts";
import { notifyStderr } from "../notifications.ts";
import { execWt, WtError } from "../process.ts";

export async function runRebranch(args: string[], ctx: ExtensionCommandContext): Promise<void> {
  const result = await execWt(["rebranch", ...injectFlag(args, "--move")], ctx.cwd);
  if (!result.cd) {
    if (result.code !== 0) throw new WtError(result);
    throw new Error("wt rebranch did not report a destination");
  }
  if (result.code === 0) notifyStderr(ctx, result);

  if (!isPersisted(ctx)) {
    if (result.code !== 0) ctx.ui.notify(result.stderr.trim(), "error");
    ctx.ui.notify(
      `Worktree moved to ${result.cd}. This in-memory session cannot move; use \`wt cd\` in a terminal.`,
      "warning",
    );
    return;
  }

  await migrateSession(ctx, result.cd, {
    ...(result.code !== 0
      ? { warning: result.stderr.trim() || `wt exited with code ${result.code}` }
      : {}),
  });
}
