import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { injectFlag } from "../args.ts";
import { notifyStderr } from "../notifications.ts";
import { runWt } from "../process.ts";
import { removeSourceAfterSessionHandoff } from "../source-removal.ts";

export async function runMerge(args: string[], ctx: ExtensionCommandContext): Promise<void> {
  const result = await runWt(["merge", ...injectFlag(args, "--defer-removal")], ctx.cwd);
  notifyStderr(ctx, result);
  const sourcePath = result.stdout.trim();
  if (!sourcePath) throw new Error("wt merge did not report the source worktree");

  await removeSourceAfterSessionHandoff(ctx, {
    sourcePath,
    destination: result.cd,
    finish: () => runWt(["rm", "--force"], sourcePath),
    inMemoryWarning:
      "Worktree merged and removed. This in-memory session cannot move; use `wt cd` in a terminal.",
  });
}
