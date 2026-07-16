import type { AutocompleteItem } from "@earendil-works/pi-tui";
import { SUBCOMMANDS, tokenizeCommandArgs, type WtSubcommand } from "./args.ts";
import { execWt, type WtResult } from "./process.ts";

const descriptions: Record<WtSubcommand, string> = {
  fork: "Fork current work to a worktree",
  checkout: "Check out a branch into a worktree",
  co: "Alias for checkout",
  rm: "Remove this worktree and return to the main one",
  merge: "Merge this worktree into the main one",
  rebranch: "Re-seat and rename the current (landed) worktree",
};

function splitCompletionPrefix(prefix: string): { words: string[]; partial: string } {
  const tokens = tokenizeCommandArgs(prefix);
  if (/\s$/.test(prefix)) return { words: tokens, partial: "" };
  return { words: tokens.slice(0, -1), partial: tokens.at(-1) ?? "" };
}

function subcommandCompletions(partial: string): AutocompleteItem[] | null {
  const items = SUBCOMMANDS.filter((command) => command.startsWith(partial)).map((command) => ({
    value: command,
    label: command,
    description: descriptions[command],
  }));
  return items.length > 0 ? items : null;
}

export function completionRequest(
  prefix: string,
): { args: string[]; words: string[] } | AutocompleteItem[] | null {
  const { words, partial } = splitCompletionPrefix(prefix);
  if (words.length === 0) return subcommandCompletions(partial);
  if (!SUBCOMMANDS.includes(words[0] as WtSubcommand)) return null;
  return { args: ["__complete", ...words, partial], words };
}

export function parseCompletions(result: WtResult, words: string[]): AutocompleteItem[] | null {
  if (result.code !== 0) return null;

  const lines = result.stdout.split("\n").filter(Boolean);
  const directiveLine = lines.at(-1);
  const directive = directiveLine?.match(/^:(\d+)$/);
  if (!directive) return null;
  lines.pop();
  if ((Number(directive[1]) & 1) !== 0) return null;

  const items = lines.map((line) => {
    const [candidate = "", description] = line.split("\t", 2);
    return {
      value: [...words, candidate].join(" "),
      label: candidate,
      ...(description ? { description } : {}),
    };
  });
  return items.length > 0 ? items : null;
}

export async function getWtCompletions(
  prefix: string,
  cwd: string,
): Promise<AutocompleteItem[] | null> {
  try {
    const request = completionRequest(prefix);
    if (!request || Array.isArray(request)) return request;
    return parseCompletions(await execWt(request.args, cwd), request.words);
  } catch {
    return null;
  }
}
