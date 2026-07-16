import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { getWtCompletions } from "./wt/autocomplete.ts";
import { createWtCommand } from "./wt/command.ts";
import { MIGRATION_NOTICE_TYPE } from "./wt/migration-notice.ts";
import { renderMigrationNotice } from "./wt/migration-renderer.ts";

export default function wtExtension(pi: ExtensionAPI): void {
  let cwd: string;

  pi.on("session_start", (_event, ctx) => {
    cwd = ctx.cwd;
  });

  pi.registerCommand(
    "wt",
    createWtCommand((prefix) => getWtCompletions(prefix, cwd)),
  );

  pi.registerMessageRenderer(MIGRATION_NOTICE_TYPE, renderMigrationNotice);
}
