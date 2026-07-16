import { describe, expect, it } from "vitest";
import { migrationNotice } from "../extensions/wt/migration-notice.ts";

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
});
