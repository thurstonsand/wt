import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { getWtCompletions } from "./wt/autocomplete.ts";
import { createWtCommand } from "./wt/command.ts";
import { createMoveCommand } from "./wt/commands/move.ts";
import { getDirectoryCompletions } from "./wt/directory-completion.ts";
import { MIGRATION_NOTICE_TYPE } from "./wt/migration-notice.ts";
import { renderMigrationNotice } from "./wt/migration-renderer.ts";
import { isWtInstalled } from "./wt/process.ts";

export default async function wtExtension(pi: ExtensionAPI): Promise<void> {
  let cwd: string;

  pi.on("session_start", (_event, ctx) => {
    cwd = ctx.cwd;
  });

  if (await isWtInstalled()) {
    pi.registerCommand(
      "wt",
      createWtCommand((prefix) => getWtCompletions(prefix, cwd)),
    );
  }
  pi.registerCommand(
    "mv",
    createMoveCommand((prefix) => getDirectoryCompletions(prefix, cwd)),
  );

  pi.registerMessageRenderer(MIGRATION_NOTICE_TYPE, renderMigrationNotice);
}
