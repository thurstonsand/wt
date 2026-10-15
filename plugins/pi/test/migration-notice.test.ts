import type { Theme } from "@earendil-works/pi-coding-agent";
import { describe, expect, it } from "vitest";
import { migrationNotice } from "../extensions/wt/migration-notice.ts";
import { renderMigrationNotice } from "../extensions/wt/migration-renderer.ts";

const migration = {
  oldCwd: "/old/worktree",
  oldPath: "/sessions/old.jsonl",
  newCwd: "/new/worktree",
  newPath: "/sessions/new.jsonl",
};

describe("migration notice", () => {
  it("describes path relocation", () => {
    expect(migrationNotice(migration)).toEqual({
      customType: "wt-session-migration",
      content:
        "Session migrated to `/new/worktree`; paths under `/old/worktree` now resolve under `/new/worktree`.",
      display: true,
      details: { destinationName: "worktree" },
    });
  });

  it("states when the old worktree was deleted", () => {
    const notice = migrationNotice(migration, "/old/worktree");
    expect(notice.content).toContain(
      "The old worktree directory `/old/worktree` has been deleted.",
    );
    expect(notice.details.deletedWorktree).toBe("/old/worktree");
  });

  it.each([0, 1])("indents by an outputPad of %i like the rest of the transcript", (outputPad) => {
    const message = { ...migrationNotice(migration), role: "custom" as const, timestamp: 0 };
    const theme = { fg: (_color: string, text: string) => text } as unknown as Theme;
    const lines = renderMigrationNotice(message, { expanded: false, outputPad }, theme)?.render(80);
    expect(lines?.[0]?.trimEnd()).toBe(`${" ".repeat(outputPad)}Switched session to worktree`);
  });
});
