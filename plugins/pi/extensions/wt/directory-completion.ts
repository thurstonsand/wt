import { readdir, stat } from "node:fs/promises";
import { homedir } from "node:os";
import { basename, dirname, join } from "node:path";
import type { AutocompleteItem } from "@earendil-works/pi-tui";
import { splitCompletionPrefix } from "./autocomplete.ts";
import { expandPath } from "./paths.ts";

function completionSearch(
  prefix: string,
  cwd: string,
  home: string,
): { directory: string; partial: string; displayBase: string } {
  if (prefix === "") {
    return { directory: expandPath(".", cwd, home), partial: "", displayBase: "" };
  }
  if (prefix === "~") {
    return { directory: expandPath(prefix, cwd, home), partial: "", displayBase: "~/" };
  }

  const expanded = expandPath(prefix, cwd, home);
  if (prefix.endsWith("/")) {
    return { directory: expanded, partial: "", displayBase: prefix };
  }

  const partial = basename(prefix);
  return {
    directory: dirname(expanded),
    partial,
    displayBase: prefix.slice(0, -partial.length),
  };
}

function quoteIfNeeded(path: string): string {
  return /\s/.test(path) ? `"${path}"` : path;
}

export async function getDirectoryCompletions(
  prefix: string,
  cwd: string,
  home: string = homedir(),
): Promise<AutocompleteItem[] | null> {
  try {
    const { words, partial } = splitCompletionPrefix(prefix);
    if (words.length > 0) return null;

    const search = completionSearch(partial, cwd, home);
    const entries = await readdir(search.directory, { withFileTypes: true });
    const items: AutocompleteItem[] = [];

    for (const entry of entries.sort((left, right) => left.name.localeCompare(right.name))) {
      if (!entry.name.toLowerCase().startsWith(search.partial.toLowerCase())) continue;

      let isDirectory = entry.isDirectory();
      if (entry.isSymbolicLink()) {
        try {
          isDirectory = (await stat(join(search.directory, entry.name))).isDirectory();
        } catch {
          isDirectory = false;
        }
      }
      if (!isDirectory) continue;

      const path = `${search.displayBase}${entry.name}/`;
      items.push({ value: quoteIfNeeded(path), label: `${entry.name}/` });
    }

    return items.length > 0 ? items : null;
  } catch {
    return null;
  }
}
