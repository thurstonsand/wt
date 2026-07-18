import type { Stats } from "node:fs";
import { stat } from "node:fs/promises";
import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import type { AutocompleteItem } from "@earendil-works/pi-tui";
import { tokenizeCommandArgs } from "../args.ts";
import { isPersisted, migrateSession } from "../migration.ts";
import { errorText } from "../notifications.ts";
import { expandPath } from "../paths.ts";

function moveDestination(input: string): string {
  const args = tokenizeCommandArgs(input);
  const destination = args[0];
  if (args.length !== 1 || !destination) throw new Error("Usage: /mv <dir>");
  return destination;
}

async function validateDirectory(destination: string): Promise<void> {
  let stats: Stats;
  try {
    stats = await stat(destination);
  } catch (error) {
    const code = (error as NodeJS.ErrnoException).code;
    if (code === "ENOENT") throw new Error(`Destination does not exist: ${destination}`);
    if (code === "ENOTDIR") throw new Error(`Destination is not a directory: ${destination}`);
    throw error;
  }

  if (!stats.isDirectory()) throw new Error(`Destination is not a directory: ${destination}`);
}

export async function runMove(input: string, ctx: ExtensionCommandContext): Promise<void> {
  const destination = expandPath(moveDestination(input), ctx.cwd);
  await validateDirectory(destination);

  if (!isPersisted(ctx)) {
    ctx.ui.notify(
      `This in-memory session cannot move to ${destination}; start pi there instead.`,
      "warning",
    );
    return;
  }

  await migrateSession(ctx, destination);
}

export function createMoveCommand(
  getArgumentCompletions: (prefix: string) => Promise<AutocompleteItem[] | null>,
) {
  return {
    description: "Move this session to another directory",
    getArgumentCompletions,
    handler: async (input: string, ctx: ExtensionCommandContext) => {
      await ctx.waitForIdle();
      try {
        await runMove(input, ctx);
      } catch (error) {
        ctx.ui.notify(errorText(error), "error");
      }
    },
  };
}
