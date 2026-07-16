import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import type { AutocompleteItem } from "@earendil-works/pi-tui";
import { parseSubcommand, tokenizeCommandArgs, type WtSubcommand } from "./args.ts";
import { runCheckout } from "./commands/checkout.ts";
import { runFork } from "./commands/fork.ts";
import { runMerge } from "./commands/merge.ts";
import { runRebranch } from "./commands/rebranch.ts";
import { runRemove } from "./commands/remove.ts";
import { errorText } from "./notifications.ts";

async function dispatch(
  command: WtSubcommand,
  args: string[],
  ctx: ExtensionCommandContext,
): Promise<void> {
  switch (command) {
    case "fork":
      await runFork(args, ctx);
      return;
    case "checkout":
    case "co":
      await runCheckout(command, args, ctx);
      return;
    case "rm":
      await runRemove(args, ctx);
      return;
    case "merge":
      await runMerge(args, ctx);
      return;
    case "rebranch":
      await runRebranch(args, ctx);
  }
}

export function createWtCommand(
  getArgumentCompletions: (prefix: string) => Promise<AutocompleteItem[] | null>,
) {
  return {
    description: "Run a session-aware wt worktree command",
    getArgumentCompletions,
    handler: async (input: string, ctx: ExtensionCommandContext) => {
      await ctx.waitForIdle();
      try {
        const { command, args } = parseSubcommand(tokenizeCommandArgs(input));
        await dispatch(command, args, ctx);
      } catch (error) {
        ctx.ui.notify(errorText(error), "error");
      }
    },
  };
}
