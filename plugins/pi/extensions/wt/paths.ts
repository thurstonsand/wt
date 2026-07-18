import { realpath } from "node:fs/promises";
import { homedir } from "node:os";
import { isAbsolute, relative, resolve, sep } from "node:path";

export function expandPath(input: string, cwd: string, home: string = homedir()): string {
  if (input === "~") return resolve(home);
  if (input.startsWith("~/")) return resolve(home, input.slice(2));
  return resolve(cwd, input);
}

export async function canonicalPath(path: string): Promise<string> {
  try {
    return await realpath(path);
  } catch {
    return resolve(path);
  }
}

export async function pathContains(parent: string, child: string): Promise<boolean> {
  const canonicalParent = await canonicalPath(parent);
  const canonicalChild = await canonicalPath(child);
  const remainder = relative(canonicalParent, canonicalChild);
  return (
    remainder === "" ||
    (remainder !== ".." && !remainder.startsWith(`..${sep}`) && !isAbsolute(remainder))
  );
}
