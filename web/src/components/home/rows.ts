import type { BrowsePage, BrowseType } from "@/lib/api/query";
import { WORK_TYPES } from "@/lib/work-types";

/** HOME_TYPES leaves out packs, which only Lumiverse reads. */
export const HOME_TYPES = WORK_TYPES.filter((type) => type !== "pack");

const ROW_MINIMUM = 6;

export interface TypeList {
  type: BrowseType;
  page: BrowsePage | null;
}

export interface HomeRow {
  type: BrowseType;
  page: BrowsePage;
}

export interface Home {
  state: "rows" | "hidden" | "few" | "empty" | "failed";
  rows: HomeRow[];
  someHidden: boolean;
}

/** homeRows keeps one row per type with at least six works to show, and names why the page has none when it has none. */
export function homeRows(lists: TypeList[]): Home {
  const loaded = lists.filter((list): list is HomeRow => list.page !== null);
  const rows = loaded.filter((list) => list.page.items.length >= ROW_MINIMUM);
  const someHidden = loaded.some((list) => list.page.suppressed > 0);
  const state = rows.length
    ? "rows"
    : !loaded.length
      ? "failed"
      : someHidden
        ? "hidden"
        : loaded.every((list) => list.page.total === 0)
          ? "empty"
          : "few";
  return { state, rows, someHidden };
}
