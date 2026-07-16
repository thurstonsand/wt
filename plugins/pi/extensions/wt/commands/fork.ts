import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { runOutbound } from "./outbound.ts";

export function runFork(args: string[], ctx: ExtensionCommandContext): Promise<void> {
  return runOutbound("fork", args, ctx);
}
