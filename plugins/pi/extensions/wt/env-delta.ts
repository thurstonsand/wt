export interface EnvDelta {
  set: Record<string, string>;
  unset: string[];
}

export interface ProbeCapture {
  before: Record<string, string>;
  after: Record<string, string>;
}

export const PROBE_SENTINEL = "__wt_env_probe_boundary__";

function parseNulEnv(chunk: string[]): Record<string, string> {
  const env: Record<string, string> = {};
  for (const entry of chunk) {
    const separator = entry.indexOf("=");
    if (separator <= 0) continue;
    env[entry.slice(0, separator)] = entry.slice(separator + 1);
  }
  return env;
}

export function parseProbeOutput(stdout: string): ProbeCapture {
  const entries = stdout.split("\0");
  const boundary = entries.indexOf(PROBE_SENTINEL);
  if (boundary === -1) throw new Error("environment probe produced no boundary marker");
  return {
    before: parseNulEnv(entries.slice(0, boundary)),
    after: parseNulEnv(entries.slice(boundary + 1)),
  };
}

export function computeEnvDelta({ before, after }: ProbeCapture): EnvDelta {
  const delta: EnvDelta = { set: {}, unset: [] };
  for (const key of new Set([...Object.keys(before), ...Object.keys(after)])) {
    const value = after[key];
    if (before[key] === value) continue;
    if (value === undefined) delta.unset.push(key);
    else delta.set[key] = value;
  }
  return delta;
}

export function applyEnvDelta(delta: EnvDelta, env: NodeJS.ProcessEnv): void {
  for (const key of delta.unset) delete env[key];
  for (const [key, value] of Object.entries(delta.set)) env[key] = value;
}
