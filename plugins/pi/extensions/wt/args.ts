import { parseArgsStringToArgv } from "string-argv";

export const SUBCOMMANDS = ["fork", "checkout", "co", "rm", "merge", "rebranch"] as const;
export type WtSubcommand = (typeof SUBCOMMANDS)[number];

export function tokenizeCommandArgs(input: string): string[] {
  return parseArgsStringToArgv(input);
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
