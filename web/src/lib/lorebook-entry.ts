import type { ChipItem } from "@/components/ui/Chip";

type ReadableEntry = {
  id?: string;
  name?: string;
  keys: string[];
  secondaryKeys?: string[];
  selective?: boolean;
  caseSensitive?: boolean;
  constant?: boolean;
  enabled: boolean;
  order?: number;
  position?: "before_character" | "after_character";
  recursion?: {
    exclude?: boolean;
    prevent?: boolean;
    delayUntil?: boolean;
  };
  text: string;
};

export type EntryNaming = "written" | "key" | "opening" | "position";

export type EntryPresentation = {
  id: string;
  position: number;
  name: string;
  named: EntryNaming;
  keys: ChipItem[];
  secondaryKeys: ChipItem[];
  firing: string[];
  note: string;
  isOff: boolean;
  text: string;
};

export type EntrySort = "book" | "name";

export type LorebookView = {
  search: string;
  sort: EntrySort;
  includeOff: boolean;
};

export type LorebookIndex = {
  total: number;
  off: number;
  entries: EntryPresentation[];
};

export function readLorebook(
  entries: readonly ReadableEntry[],
  view: LorebookView,
): LorebookIndex {
  const book = { showsOrder: ordersDiffer(entries) };
  const read = entries.map((entry, index) => readEntry(entry, index + 1, book));
  const wanted = read.filter(
    (entry) => (view.includeOff || !entry.isOff) && matches(entry, view.search),
  );

  return {
    total: entries.length,
    off: read.filter((entry) => entry.isOff).length,
    entries: view.sort === "name" ? byName(wanted) : wanted,
  };
}

type BookContext = {
  showsOrder: boolean;
};

export function readEntry(
  entry: ReadableEntry,
  position: number,
  book: BookContext = { showsOrder: false },
): EntryPresentation {
  const keys = chips(entry.keys);
  const secondaryKeys =
    entry.selective === true ? chips(entry.secondaryKeys ?? []) : [];
  const constant = entry.constant === true;

  const named = nameFor(entry, position, keys);

  return {
    id: entry.id ?? `entry-${position}`,
    position,
    name: named.name,
    named: named.named,
    keys,
    secondaryKeys,
    firing: firingRules(entry, keys.length, secondaryKeys.length, book),
    note: indexNote(entry.enabled, constant, keys.length),
    isOff: !entry.enabled,
    text: entry.text,
  };
}

function nameFor(
  entry: ReadableEntry,
  position: number,
  keys: readonly ChipItem[],
): { name: string; named: EntryNaming } {
  const written = entry.name?.trim();
  if (written) return { name: written, named: "written" };
  if (keys.length > 0) return { name: keys[0].label, named: "key" };
  const first = opening(entry.text);
  if (first) return { name: first, named: "opening" };
  return { name: `Entry ${position}`, named: "position" };
}

const NAME_LENGTH = 54;

const MARKDOWN = /(\*\*|\*|__|`|^#{1,6}\s+|^>\s+)/gm;

function opening(text: string): string {
  const line = text.replace(MARKDOWN, "").replace(/\s+/g, " ").trim();
  if (line.length <= NAME_LENGTH) return line;
  const cut = line.slice(0, NAME_LENGTH);
  const lastSpace = cut.lastIndexOf(" ");
  const words = lastSpace > NAME_LENGTH / 2 ? cut.slice(0, lastSpace) : cut;
  return `${words.trimEnd()}\u2026`;
}

function chips(keys: readonly string[]): ChipItem[] {
  return keys
    .map((key, index) => ({ id: `${index}-${key.trim()}`, label: key.trim() }))
    .filter((chip) => chip.label !== "");
}

function indexNote(
  enabled: boolean,
  constant: boolean,
  keyCount: number,
): string {
  if (!enabled) return "Off";
  if (constant) return "Always on";
  if (keyCount === 0) return "No keys";
  return keyCount === 1 ? "1 key" : `${keyCount} keys`;
}

function firingRules(
  entry: ReadableEntry,
  keyCount: number,
  secondaryCount: number,
  book: BookContext,
): string[] {
  const rules: string[] = [];
  if (!entry.enabled) rules.push("Switched off");
  if (entry.constant === true) rules.push("Always on");
  else if (keyCount === 0) rules.push("No keys, so nothing switches it on");
  else if (keyCount === 1) rules.push("Switched on by its key");
  else rules.push(`Switched on by any of its ${keyCount} keys`);

  if (secondaryCount > 0) rules.push("Needs a second key as well");
  if (entry.caseSensitive === true) rules.push("Matches the case of its keys");
  if (entry.position === "before_character") {
    rules.push("Goes in before the character");
  }
  if (entry.position === "after_character") {
    rules.push("Goes in after the character");
  }
  if (book.showsOrder && entry.order != null) {
    rules.push(`Order ${entry.order}`);
  }
  if (entry.recursion?.exclude) {
    rules.push("Other entries cannot switch it on");
  }
  if (entry.recursion?.prevent) rules.push("Cannot switch other entries on");
  if (entry.recursion?.delayUntil) rules.push("Held back until a later pass");
  return rules;
}

function ordersDiffer(entries: readonly ReadableEntry[]): boolean {
  const orders = new Set(entries.map((entry) => entry.order));
  return orders.size > 1;
}

function matches(entry: EntryPresentation, search: string): boolean {
  const wanted = search.trim().toLocaleLowerCase();
  if (wanted === "") return true;
  if (entry.name.toLocaleLowerCase().includes(wanted)) return true;
  return [...entry.keys, ...entry.secondaryKeys].some((key) =>
    key.label.toLocaleLowerCase().includes(wanted),
  );
}

function byName(entries: readonly EntryPresentation[]): EntryPresentation[] {
  return [...entries].sort(
    (one, other) =>
      one.name.localeCompare(other.name) || one.position - other.position,
  );
}
