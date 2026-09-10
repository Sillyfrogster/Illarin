"use client";

import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/cn";
import { chosenIndex, itemKeys, keyAfterMove } from "./collection";
import {
  AddAction,
  ItemMoveActions,
  Note,
  RemoveAction,
  TextField,
} from "./fields";

export type CollectionRow = {
  detail?: string;
  id?: string;
  name: string;
  off?: boolean;
  search: string;
  sealed?: boolean;
};

/** A collection is its list or one of its items, never both at once, so each gets the whole rail. */
export function CollectionStep({
  above,
  children,
  chosen,
  emptyMessage,
  noun,
  onAdd,
  onChoose,
  onMove,
  onRemove,
  pending,
  plural,
  rows,
}: {
  above?: ReactNode;
  children: (index: number) => ReactNode;
  chosen: string | null;
  emptyMessage: string;
  noun: string;
  onAdd: () => void;
  onChoose: (key: string | null) => void;
  onMove?: (from: number, to: number) => void;
  onRemove: (index: number) => void;
  pending: boolean;
  plural?: string;
  rows: CollectionRow[];
}) {
  const keys = useMemo(() => itemKeys(rows), [rows]);
  const opened = useRef<string | null>(null);
  const open = chosen !== null;
  const index = chosenIndex(keys, chosen);
  const nouns = plural ?? `${noun}s`;
  if (chosen !== null) opened.current = chosen;

  if (open && index !== -1) {
    return (
      <div className="flex flex-col gap-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <button
            className="-ml-2 inline-flex min-h-11 items-center gap-1 rounded-control pr-3 pl-1 text-meta font-medium text-accent outline-offset-3 hover:underline"
            onClick={() => onChoose(null)}
            type="button"
          >
            <ChevronLeft aria-hidden="true" size={16} />
            All {rows.length} {nouns}
          </button>
          <p className="text-label text-mute">
            {index + 1} of {rows.length}
          </p>
        </div>
        <h3 className="font-display text-section font-medium text-ink wrap-anywhere">
          {rows[index].name}
        </h3>
        {children(index)}
        <div className="flex flex-wrap items-center gap-1 pt-1">
          {onMove ? (
            <ItemMoveActions
              moves={{
                onEarlier: () => {
                  onMove(index, index - 1);
                  onChoose(keyAfterMove(keys[index], index - 1));
                },
                onLater: () => {
                  onMove(index, index + 1);
                  onChoose(keyAfterMove(keys[index], index + 1));
                },
                position: index,
                total: rows.length,
              }}
              pending={pending}
            />
          ) : null}
          <RemoveAction
            disabled={pending}
            onClick={() => {
              onRemove(index);
              onChoose(null);
            }}
          >
            Remove {noun}
          </RemoveAction>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      {above}
      <CollectionList
        emptyMessage={emptyMessage}
        keys={keys}
        lastOpened={opened.current}
        noun={noun}
        nouns={nouns}
        onAdd={() => {
          onAdd();
          onChoose(`new:${rows.length}`);
        }}
        onChoose={onChoose}
        pending={pending}
        rows={rows}
      />
    </div>
  );
}

function CollectionList({
  emptyMessage,
  keys,
  lastOpened,
  noun,
  nouns,
  onAdd,
  onChoose,
  pending,
  rows,
}: {
  emptyMessage: string;
  keys: string[];
  lastOpened: string | null;
  noun: string;
  nouns: string;
  onAdd: () => void;
  onChoose: (key: string | null) => void;
  pending: boolean;
  rows: CollectionRow[];
}) {
  const [search, setSearch] = useState("");
  const showRow = useCallback((row: HTMLButtonElement | null) => {
    row?.scrollIntoView({ block: "center" });
  }, []);
  const searchId = useId();
  const wanted = search.trim().toLowerCase();
  const matching = useMemo(
    () =>
      rows
        .map((row, index) => ({ index, row }))
        .filter(({ row }) => wanted === "" || row.search.includes(wanted)),
    [rows, wanted],
  );

  return (
    <div className="flex flex-col gap-4">
      {rows.length > 6 ? (
        <div className="relative">
          <label className="sr-only" htmlFor={searchId}>
            Search the {nouns}
          </label>
          <Search
            aria-hidden="true"
            className="pointer-events-none absolute top-3.5 left-3 size-4 text-mute"
            size={16}
          />
          <TextField
            className="pl-9"
            id={searchId}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={`Search the ${nouns}`}
            type="search"
            value={search}
          />
        </div>
      ) : null}

      <AddAction disabled={pending} onClick={onAdd}>
        Add {noun}
      </AddAction>

      {rows.length === 0 ? <Note>{emptyMessage}</Note> : null}

      {wanted === "" ? null : (
        <Note>
          {matching.length === 0
            ? `No ${noun} matches that.`
            : `${matching.length} of ${rows.length} ${nouns}`}
        </Note>
      )}

      {matching.length > 0 ? (
        <ol className="-mx-2 flex list-none flex-col gap-0.5">
          {matching.map(({ index, row }) => (
            <li key={keys[index]}>
              <button
                className="flex w-full items-center gap-3 rounded-control px-2 py-2.5 text-left outline-offset-3 hover:bg-deep"
                onClick={() => onChoose(keys[index])}
                ref={keys[index] === lastOpened ? showRow : undefined}
                type="button"
              >
                <span className="min-w-0 flex-1">
                  <span className="block text-ui font-medium text-ink wrap-anywhere">
                    {row.name}
                  </span>
                  {row.detail ? (
                    <span className="mt-0.5 block text-meta text-mute wrap-anywhere">
                      {row.detail}
                    </span>
                  ) : null}
                  {row.off || row.sealed ? (
                    <span className="mt-1.5 flex flex-wrap gap-1.5">
                      {row.off ? <RowBadge>Switched off</RowBadge> : null}
                      {row.sealed ? <RowBadge accent>Sealed</RowBadge> : null}
                    </span>
                  ) : null}
                </span>
                <ChevronRight
                  aria-hidden="true"
                  className="size-4 shrink-0 text-mute"
                  size={16}
                />
              </button>
            </li>
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function RowBadge({
  accent,
  children,
}: {
  accent?: boolean;
  children: ReactNode;
}) {
  return (
    <span
      className={cn(
        "rounded-control px-2 py-0.5 text-label font-medium",
        accent ? "bg-accent-wash text-accent" : "bg-deep text-mute",
      )}
    >
      {children}
    </span>
  );
}
