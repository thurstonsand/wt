import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import { runOutbound } from "./outbound.ts";

export function runCheckout(
  command: "checkout" | "co",
  args: string[],
  ctx: ExtensionCommandContext,
): Promise<void> {
  return runOutbound(command, args, ctx);
}
