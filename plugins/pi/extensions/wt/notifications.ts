import type { ExtensionCommandContext } from "@earendil-works/pi-coding-agent";
import type { WtResult } from "./process.ts";

export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function notifyStderr(ctx: ExtensionCommandContext, result: WtResult): void {
  const stderr = result.stderr.trim();
  if (stderr) ctx.ui.notify(stderr, "warning");
}

export function notifyResult(ctx: ExtensionCommandContext, result: WtResult): void {
  const stdout = result.stdout.trim();
  if (stdout) ctx.ui.notify(stdout, "info");
  notifyStderr(ctx, result);
}
