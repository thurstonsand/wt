import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  applyEnvDelta,
  computeEnvDelta,
  PROBE_SENTINEL,
  parseProbeOutput,
} from "../extensions/wt/env-delta.ts";
import { probeScript, reconcileDirectoryEnv } from "../extensions/wt/env-probe.ts";

function probeOutput(before: string[], after: string[]): string {
  return `${before.map((entry) => `${entry}\0`).join("")}${PROBE_SENTINEL}\0${after
    .map((entry) => `${entry}\0`)
    .join("")}`;
}

describe("environment probe output", () => {
  it("splits captures at the sentinel and keeps values containing = and newlines", () => {
    const capture = parseProbeOutput(
      probeOutput(["PATH=/usr/bin", "MULTI=first\nsecond", "FLAGS=-a=b"], ["PATH=/opt/bin"]),
    );

    expect(capture.before).toEqual({
      PATH: "/usr/bin",
      MULTI: "first\nsecond",
      FLAGS: "-a=b",
    });
    expect(capture.after).toEqual({ PATH: "/opt/bin" });
  });

  it("rejects output without a sentinel", () => {
    expect(() => parseProbeOutput("PATH=/usr/bin\0")).toThrow(/boundary marker/);
  });
});

describe("environment delta", () => {
  it("reports additions, changes, and removals", () => {
    const delta = computeEnvDelta({
      before: { PATH: "/usr/bin", GONE: "1", SAME: "keep" },
      after: { PATH: "/opt/bin", ADDED: "2", SAME: "keep" },
    });

    expect(delta).toEqual({
      set: { PATH: "/opt/bin", ADDED: "2" },
      unset: ["GONE"],
    });
  });

  it("ignores per-invocation values that the single probe shell repeats", () => {
    const delta = computeEnvDelta({
      before: {
        STARSHIP_SESSION_KEY: "same",
        __MISE_SESSION: "same",
        SHLVL: "3",
      },
      after: {
        STARSHIP_SESSION_KEY: "same",
        __MISE_SESSION: "same",
        SHLVL: "3",
      },
    });

    expect(delta).toEqual({ set: {}, unset: [] });
  });

  it("carries PWD over, since it names where the session now lives", () => {
    const delta = computeEnvDelta({
      before: { PWD: "/old" },
      after: { PWD: "/new" },
    });

    expect(delta).toEqual({ set: { PWD: "/new" }, unset: [] });
  });

  it("mutates only the delta's keys when applied", () => {
    const env: NodeJS.ProcessEnv = {
      PATH: "/usr/bin",
      GONE: "1",
      UNTOUCHED: "keep",
    };
    applyEnvDelta({ set: { PATH: "/opt/bin", ADDED: "2" }, unset: ["GONE"] }, env);

    expect(env).toEqual({ PATH: "/opt/bin", ADDED: "2", UNTOUCHED: "keep" });
  });
});

describe("probe script", () => {
  it("fires zsh's chpwd and precmd hooks", () => {
    const script = probeScript("/bin/zsh", "/tmp/dest");
    expect(script).toContain("cd -- '/tmp/dest'");
    expect(script).toContain("$chpwd_functions $precmd_functions");
  });

  it("fires bash's PROMPT_COMMAND in both its string and array shapes", () => {
    // biome-ignore lint/suspicious/noTemplateCurlyInString: shell expansion, not a JS template
    expect(probeScript("/usr/local/bin/bash", "/tmp/dest")).toContain('"${PROMPT_COMMAND[@]}"');
  });

  it("quotes destinations containing quotes", () => {
    expect(probeScript("/bin/zsh", "/tmp/it's here")).toContain(`cd -- '/tmp/it'\\''s here'`);
  });

  it("skips shells without known directory hooks", () => {
    expect(probeScript("/usr/bin/fish", "/tmp/dest")).toBeUndefined();
  });
});

describe("directory environment reconciliation", () => {
  afterEach(() => {
    delete process.env.WT_PROBE_APPLIED;
    delete process.env.WT_PROBE_STALE;
  });

  it("does nothing without a usable login shell", async () => {
    await expect(reconcileDirectoryEnv("/tmp", "/tmp", "/usr/bin/fish")).resolves.toBeUndefined();
    await expect(reconcileDirectoryEnv("/tmp", "/tmp", "")).resolves.toBeUndefined();
  });

  it("applies what a destination's shell hook exports", async () => {
    const root = await mkdtemp(join(tmpdir(), "pi-wt-env-"));
    const destination = join(root, "destination");
    await mkdir(destination);
    const shell = join(root, "zsh");
    await writeFile(
      shell,
      `#!/bin/zsh\nchpwd_functions=(wt_probe_hook)\nwt_probe_hook() { export WT_PROBE_APPLIED="$PWD"; unset WT_PROBE_STALE }\neval "$2"\n`,
      { mode: 0o755 },
    );

    process.env.WT_PROBE_STALE = "1";
    const delta = await reconcileDirectoryEnv(root, destination, shell);

    expect(delta?.set.WT_PROBE_APPLIED).toBe(destination);
    expect(delta?.unset).toContain("WT_PROBE_STALE");
    expect(process.env.WT_PROBE_APPLIED).toBe(destination);
    expect(process.env.WT_PROBE_STALE).toBeUndefined();
  });

  it("fails loudly enough to be caught when the shell cannot run", async () => {
    await expect(reconcileDirectoryEnv("/tmp", "/tmp", "/nonexistent/zsh")).rejects.toThrow();
  });
});
