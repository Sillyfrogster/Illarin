const UNSAVED = "new:";

export function itemKeys(items: readonly { id?: string }[]): string[] {
  return items.map((item, index) => item.id ?? `${UNSAVED}${index}`);
}

export function chosenIndex(keys: readonly string[], chosen: string | null) {
  if (keys.length === 0) return -1;
  if (chosen === null) return 0;
  const found = keys.indexOf(chosen);
  if (found !== -1) return found;
  if (!chosen.startsWith(UNSAVED)) return 0;
  const added = Number(chosen.slice(UNSAVED.length));
  return Number.isInteger(added) && added < keys.length ? added : 0;
}

export function keyAfterMove(key: string, to: number): string {
  return key.startsWith(UNSAVED) ? `${UNSAVED}${to}` : key;
}

export function moveItem<T>(
  items: readonly T[],
  from: number,
  to: number,
): T[] {
  if (to < 0 || to >= items.length || from === to) return [...items];
  const next = [...items];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
}

export function replaceAt<T>(
  items: readonly T[],
  index: number,
  changes: Partial<T>,
): T[] {
  return items.map((item, position) =>
    position === index ? { ...item, ...changes } : item,
  );
}

export function without<T>(items: readonly T[], index: number): T[] {
  return items.filter((_, position) => position !== index);
}

export function readLines(written: string): string[] {
  return written
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");
}

export function writeLines(lines: readonly string[] | undefined): string {
  return (lines ?? []).join("\n");
}
