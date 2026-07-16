import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { injectFlag } from "../args.ts";
import { runWt } from "../process.ts";
import { removeSourceAfterSessionHandoff } from "../source-removal.ts";

export async function runRemove(args: string[], ctx: ExtensionCommandContext): Promise<void> {
  const validation = await runWt(["rm", ...injectFlag(args, "--validate-only")], ctx.cwd);
  const sourcePath = validation.stdout.trim();
  if (!sourcePath) throw new Error("wt rm did not report the source worktree");

  await removeSourceAfterSessionHandoff(ctx, {
    sourcePath,
    destination: validation.cd,
    finish: () => runWt(["rm", ...args], sourcePath),
    inMemoryWarning:
      "Worktree removed. This in-memory session cannot move; use `wt cd` in a terminal.",
  });
}
