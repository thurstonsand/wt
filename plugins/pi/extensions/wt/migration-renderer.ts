import type { MessageRenderer } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import type { MigrationNoticeDetails } from "./migration-notice.ts";

export const renderMigrationNotice: MessageRenderer<MigrationNoticeDetails> = (
  message,
  _options,
  theme,
) => {
  if (!message.details) return undefined;
  const removed = message.details.deletedWorktree ? " · removed old worktree" : "";
  return new Text(
    theme.fg("dim", `Switched session to ${message.details.destinationName}${removed}`),
    0,
    0,
  );
};
