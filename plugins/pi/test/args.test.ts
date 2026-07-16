import { describe, expect, it } from "vitest";
import { injectFlag, parseSubcommand, tokenizeCommandArgs } from "../extensions/wt/args.ts";
import { completionRequest, parseCompletions } from "../extensions/wt/autocomplete.ts";

describe("command arguments", () => {
  it("preserves quoted flag values", () => {
    expect(tokenizeCommandArgs('fork feature --with "config/*.local env"')).toEqual([
      "fork",
      "feature",
      "--with",
      "config/*.local env",
    ]);
  });

  it("rejects commands that do not migrate sessions", () => {
    expect(() => parseSubcommand(["list"])).toThrow("Unsupported /wt command: list");
  });

  it("injects integration flags exactly once", () => {
    expect(injectFlag(["next", "--move"], "--move")).toEqual(["next", "--move"]);
    expect(injectFlag(["next", "--move=false"], "--move")).toEqual(["next", "--move"]);
  });
});

describe("completion", () => {
  it("offers only session-aware subcommands", () => {
    expect(completionRequest("r")).toEqual([
      expect.objectContaining({ value: "rm" }),
      expect.objectContaining({ value: "rebranch" }),
    ]);
  });

  it("builds cobra completion requests from the current partial", () => {
    expect(completionRequest("checkout feat")).toEqual({
      args: ["__complete", "checkout", "feat"],
      words: ["checkout"],
    });
  });

  it("reconstructs the full command argument value", () => {
    const items = parseCompletions(
      {
        code: 0,
        stderr: "",
        stdout: "feature/one\tbranch description\nfeature/two\n:4\n",
      },
      ["co"],
    );
    expect(items).toEqual([
      {
        value: "co feature/one",
        label: "feature/one",
        description: "branch description",
      },
      { value: "co feature/two", label: "feature/two" },
    ]);
  });
});
