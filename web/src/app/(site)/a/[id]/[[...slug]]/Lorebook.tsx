"use client";

import { Search } from "lucide-react";
import {
  type KeyboardEvent,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { ChipSet } from "@/components/ui/Chip";
import { RichText } from "@/components/ui/RichText";
import type { LorebookEntry } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  type EntryPresentation,
  type EntrySort,
  type LorebookIndex,
  readLorebook,
} from "@/lib/lorebook-entry";

const KEY_PREVIEW_LIMIT = 6;

export function Lorebook({ entries }: { entries: LorebookEntry[] }) {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<EntrySort>("book");
  const [includeOff, setIncludeOff] = useState(true);
  const [chosen, setChosen] = useState<string | null>(null);
  const names = useId();

  const book = useMemo(
    () => readLorebook(entries, { search, sort, includeOff }),
    [entries, search, sort, includeOff],
  );

  const shown =
    book.entries.find((entry) => entry.id === chosen) ?? book.entries[0];

  return (
    <div className="flex min-w-0 flex-col gap-5 [--index-gap:34px] [--index-width:272px]">
      <Controls
        search={search}
        sort={sort}
        includeOff={includeOff}
        book={book}
        onSearch={setSearch}
        onSort={setSort}
        onIncludeOff={setIncludeOff}
      />
      {book.entries.length === 0 ? (
        <Nothing search={search} book={book} />
      ) : (
        <div className="relative grid items-start gap-6 @min-[760px]:grid-cols-[var(--index-width)_minmax(0,1fr)] @min-[760px]:gap-[var(--index-gap)] @min-[760px]:before:absolute @min-[760px]:before:inset-y-0 @min-[760px]:before:left-[calc(var(--index-width)+var(--index-gap)/2)] @min-[760px]:before:border-rule @min-[760px]:before:border-l @min-[760px]:before:content-['']">
          <Index
            entries={book.entries}
            names={names}
            shownId={shown?.id}
            onChoose={setChosen}
          />
          {shown ? (
            <Entry entry={shown} labelledBy={`${names}-${shown.id}`} />
          ) : null}
        </div>
      )}
    </div>
  );
}

function Index({
  entries,
  names,
  shownId,
  onChoose,
}: {
  entries: EntryPresentation[];
  names: string;
  shownId: string | undefined;
  onChoose: (id: string) => void;
}) {
  const index = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const list = index.current;
    const row = shownId
      ? list?.querySelector(`[data-row="${CSS.escape(shownId)}"]`)
      : null;
    if (!list || !row) return;
    const rows = list.getBoundingClientRect();
    const chosenRow = row.getBoundingClientRect();
    if (chosenRow.top < rows.top) {
      list.scrollTop -= rows.top - chosenRow.top;
    } else if (chosenRow.bottom > rows.bottom) {
      list.scrollTop += chosenRow.bottom - rows.bottom;
    }
  }, [shownId]);

  function moveFocus(event: KeyboardEvent<HTMLButtonElement>, from: number) {
    const to = nextRow(event.key, from, entries.length);
    if (to === null) return;
    event.preventDefault();
    index.current?.querySelectorAll("button")[to]?.focus();
  }

  return (
    <div
      className="flex max-h-67 flex-col gap-px overflow-y-auto overscroll-contain border-rule border-b pr-1 pb-5 [scrollbar-color:var(--v-rule)_transparent] [scrollbar-width:thin] @min-[760px]:sticky @min-[760px]:top-[calc(var(--header-height)+22px)] @min-[760px]:max-h-[min(70dvh,616px)] @min-[760px]:border-b-0 @min-[760px]:pr-3 @min-[760px]:pb-0"
      ref={index}
      role="tablist"
      aria-label="The entries in this book"
      aria-orientation="vertical"
    >
      {entries.map((entry, row) => (
        <button
          key={entry.id}
          type="button"
          role="tab"
          id={`${names}-${entry.id}`}
          aria-selected={entry.id === shownId}
          tabIndex={entry.id === shownId ? 0 : -1}
          className="flex w-full cursor-pointer items-baseline gap-2.5 rounded-control px-3 py-2.5 text-left outline-offset-[-1px] hover:bg-deep/60 aria-selected:bg-deep aria-selected:shadow-[inset_2px_0_0_var(--v-action)]"
          data-row={entry.id}
          data-off={entry.isOff ? true : undefined}
          onFocus={() => onChoose(entry.id)}
          onClick={() => onChoose(entry.id)}
          onKeyDown={(event) => moveFocus(event, row)}
        >
          <span
            className={cn(
              "min-w-0 flex-1 truncate text-ui font-medium",
              entry.isOff ? "text-mute" : "text-ink",
            )}
            title={entry.name}
          >
            {entry.name}
          </span>
          <span className="shrink-0 text-label text-mute tabular-nums">
            {entry.note}
          </span>
        </button>
      ))}
    </div>
  );
}

function Nothing({ search, book }: { search: string; book: LorebookIndex }) {
  const wanted = search.trim();
  if (wanted !== "") {
    return (
      <p className="!text-ui text-mute">
        Nothing here is named <strong className="text-ink">{wanted}</strong>,
        and no key holds it.
      </p>
    );
  }
  return (
    <p className="!text-ui text-mute">
      {book.total === 0
        ? "This book holds no entries yet."
        : "Every entry in this book is switched off."}
    </p>
  );
}

function Controls({
  search,
  sort,
  includeOff,
  book,
  onSearch,
  onSort,
  onIncludeOff,
}: {
  search: string;
  sort: EntrySort;
  includeOff: boolean;
  book: LorebookIndex;
  onSearch: (search: string) => void;
  onSort: (sort: EntrySort) => void;
  onIncludeOff: (includeOff: boolean) => void;
}) {
  const searchField = useId();
  const sortField = useId();

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2.5">
      <div className="flex min-w-0 flex-[0_1_340px] items-center gap-2.5 rounded-control bg-deep px-3 text-mute">
        <Search size={16} aria-hidden="true" />
        <label className="sr-only" htmlFor={searchField}>
          Search the entries by name and by key
        </label>
        <input
          className="min-h-11 min-w-0 flex-auto border-0 bg-transparent text-ui text-ink outline-offset-3"
          id={searchField}
          type="search"
          value={search}
          placeholder="Search names and keys"
          onChange={(event) => onSearch(event.target.value)}
        />
      </div>
      <div className="flex items-center gap-2">
        <label className="text-label text-mute" htmlFor={sortField}>
          Order
        </label>
        <select
          className="min-h-11 rounded-control bg-deep px-3 text-meta text-ink outline-offset-3"
          id={sortField}
          value={sort}
          onChange={(event) => onSort(event.target.value as EntrySort)}
        >
          <option value="book">As the book holds them</option>
          <option value="name">By name, A to Z</option>
        </select>
      </div>
      {book.off > 0 ? (
        <label className="inline-flex min-h-11 items-center gap-2 text-label text-mute">
          <input
            type="checkbox"
            checked={includeOff}
            className="size-4 accent-[var(--v-action)]"
            onChange={(event) => onIncludeOff(event.target.checked)}
          />
          Include the {book.off} that {book.off === 1 ? "is" : "are"} off
        </label>
      ) : null}
      <p className="text-label text-mute @min-[760px]:ml-auto">
        {book.entries.length === book.total
          ? null
          : `${book.entries.length} of ${book.total}`}
      </p>
    </div>
  );
}

// biome-ignore-start lint/a11y/noNoninteractiveTabindex: The panel scrolls by keyboard.
function Entry({
  entry,
  labelledBy,
}: {
  entry: EntryPresentation;
  labelledBy: string;
}) {
  return (
    <div
      className="flex min-w-0 flex-col gap-3 outline-offset-8"
      role="tabpanel"
      aria-labelledby={labelledBy}
      tabIndex={0}
    >
      {entry.named === "opening" ? null : (
        <h4 className="font-display text-section font-medium tracking-tight text-ink [overflow-wrap:anywhere]">
          {entry.name}
        </h4>
      )}
      <p className="!text-label text-mute">{entry.firing.join(" · ")}</p>
      {entry.keys.length > 0 ? (
        <ChipSet
          className="mt-0.5"
          limit={KEY_PREVIEW_LIMIT}
          items={entry.keys}
        />
      ) : null}
      {entry.secondaryKeys.length > 0 ? (
        <div className="flex flex-wrap items-baseline gap-x-2.5 gap-y-1.5">
          <p className="!text-label text-mute">Second keys</p>
          <ChipSet limit={KEY_PREVIEW_LIMIT} items={entry.secondaryKeys} />
        </div>
      ) : null}
      <RichText className="mt-1.5 max-w-[70ch]" text={entry.text} />
    </div>
  );
}
// biome-ignore-end lint/a11y/noNoninteractiveTabindex: End panel exception.

function nextRow(key: string, from: number, rows: number): number | null {
  if (rows === 0) return null;
  if (key === "ArrowDown") return Math.min(from + 1, rows - 1);
  if (key === "ArrowUp") return Math.max(from - 1, 0);
  if (key === "Home") return 0;
  if (key === "End") return rows - 1;
  return null;
}
