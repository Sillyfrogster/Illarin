import type { ReactNode } from "react";

/** One named thing in a collection, listed by name and read on its own. */
export type CollectionItem = {
  key: string;
  name: string;
  note?: string;
  group?: string;
  off?: boolean;
  terms?: readonly string[];
  detail: ReactNode;
};

export type CollectionOrder = "given" | "name";

export type CollectionView = {
  search: string;
  order: CollectionOrder;
  includeOff: boolean;
};

/** Past this many named items a run is longer than a page wants to carry. */
export const BROWSE_THRESHOLD = 16;

/** A run earns a browser once it is long and its items answer to names. */
export function browsable(items: readonly CollectionItem[]): boolean {
  if (items.length <= BROWSE_THRESHOLD) return false;
  return items.every((item) => item.name.trim() !== "");
}

/** viewCollection narrows and orders the items the way a reader asked. */
export function viewCollection(
  items: readonly CollectionItem[],
  view: CollectionView,
): CollectionItem[] {
  const wanted = view.search.trim().toLocaleLowerCase();
  const shown = items.filter(
    (item) =>
      (view.includeOff || !item.off) &&
      (wanted === "" || matches(item, wanted)),
  );
  if (view.order !== "name") return shown;
  return shown
    .map((item, position) => ({ item, position }))
    .sort(
      (one, other) =>
        one.item.name.localeCompare(other.item.name) ||
        one.position - other.position,
    )
    .map(({ item }) => item);
}

function matches(item: CollectionItem, wanted: string): boolean {
  return [item.name, item.group ?? "", ...(item.terms ?? [])].some((term) =>
    term.toLocaleLowerCase().includes(wanted),
  );
}

/** countOf names a total in the collection's own noun. */
export function countOf(total: number, noun: string): string {
  return `${total} ${total === 1 ? noun : `${noun}s`}`;
}
