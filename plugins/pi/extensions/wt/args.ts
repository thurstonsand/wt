export const SUBCOMMANDS = ["fork", "checkout", "co", "rm", "merge", "rebranch"] as const;
export type WtSubcommand = (typeof SUBCOMMANDS)[number];

export function tokenizeCommandArgs(input: string): string[] {
  const args: string[] = [];
  let current = "";
  let quote: "'" | '"' | undefined;
  let started = false;

  for (const character of input) {
    if (quote) {
      if (character === quote) quote = undefined;
      else current += character;
    } else if (character === "'" || character === '"') {
      quote = character;
      started = true;
    } else if (/\s/.test(character)) {
      if (started) {
        args.push(current);
        current = "";
        started = false;
      }
    } else {
      current += character;
      started = true;
    }
  }

  if (started) args.push(current);
  return args;
}

export function parseSubcommand(args: string[]): { command: WtSubcommand; args: string[] } {
  const command = args[0];
  if (!SUBCOMMANDS.includes(command as WtSubcommand)) {
    const shown = command ? ` ${command}` : "";
    throw new Error(
      `Unsupported /wt command:${shown || " (missing)"}. Use: ${SUBCOMMANDS.join(", ")}`,
    );
  }
  return { command: command as WtSubcommand, args: args.slice(1) };
}

export function injectFlag(args: string[], flag: string): string[] {
  const separator = args.indexOf("--");
  const options = separator < 0 ? args : args.slice(0, separator);
  const positionals = separator < 0 ? [] : args.slice(separator);
  const filtered = options.filter(
    (argument) => argument !== flag && !argument.startsWith(`${flag}=`),
  );
  return [...filtered, flag, ...positionals];
}
