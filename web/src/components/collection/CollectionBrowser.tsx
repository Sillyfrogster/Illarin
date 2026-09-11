"use client";

import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowLeft, Search } from "lucide-react";
import {
  type KeyboardEvent,
  type ReactNode,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/cn";
import {
  type CollectionItem,
  type CollectionOrder,
  viewCollection,
} from "@/lib/collection";

type Row =
  | { kind: "heading"; group: string; key: string }
  | { kind: "item"; item: CollectionItem; key: string };

const HEADING_HEIGHT = 34;
const ITEM_HEIGHT = 44;

/** CollectionBrowser opens a long run in a plate of its own, indexed and searchable. */
export function CollectionBrowser({
  count,
  items,
  noun,
  onOpenChange,
  open,
  returnTo,
  start,
  title,
}: {
  count: string;
  items: readonly CollectionItem[];
  noun: string;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  returnTo?: HTMLElement | null;
  start?: string | null;
  title: string;
}) {
  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        aria-describedby={undefined}
        className="w-full"
        onCloseAutoFocus={(event) => {
          if (!returnTo) return;
          event.preventDefault();
          returnTo.focus();
        }}
      >
        {open ? (
          <Browser
            count={count}
            items={items}
            noun={noun}
            start={start}
            title={title}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function Browser({
  count,
  items,
  noun,
  start,
  title,
}: {
  count: string;
  items: readonly CollectionItem[];
  noun: string;
  start?: string | null;
  title: string;
}) {
  const [search, setSearch] = useState("");
  const [order, setOrder] = useState<CollectionOrder>("given");
  const [includeOff, setIncludeOff] = useState(true);
  const [chosen, setChosen] = useState<string | null>(start ?? null);
  const [pane, setPane] = useState<"index" | "detail">(
    start ? "detail" : "index",
  );
  const names = useId();

  const shown = useMemo(
    () => viewCollection(items, { includeOff, order, search }),
    [items, includeOff, order, search],
  );
  const rows = useMemo(() => rowsOf(shown, order === "given"), [shown, order]);
  const off = items.filter((item) => item.off).length;
  const current = shown.find((item) => item.key === chosen) ?? shown[0] ?? null;

  function choose(key: string) {
    setChosen(key);
    setPane("detail");
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex flex-col gap-4 px-5 pt-5 pr-16 sm:px-7 sm:pt-6">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <DialogTitle className="font-display text-title font-medium tracking-tight text-ink">
            {title}
          </DialogTitle>
          <DialogDescription className="font-ui text-meta text-mute">
            {count}
          </DialogDescription>
        </div>
        <Controls
          includeOff={includeOff}
          noun={noun}
          off={off}
          onIncludeOff={setIncludeOff}
          onOrder={setOrder}
          onSearch={setSearch}
          order={order}
          search={search}
          shown={shown.length}
          total={items.length}
        />
      </header>

      <div
        className="mt-4 grid min-h-0 flex-1 grid-cols-1 border-rule border-t sm:mt-5 sm:grid-cols-[minmax(240px,300px)_minmax(0,1fr)]"
        data-pane={pane}
      >
        <div
          className={cn(
            "min-h-0 sm:block sm:border-rule sm:border-r",
            pane === "detail" && "hidden",
          )}
        >
          {rows.length === 0 ? (
            <p className="px-5 py-6 font-ui text-ui text-mute sm:px-7">
              {search.trim()
                ? `Nothing here is called ${search.trim()}.`
                : `All ${noun} are off.`}
            </p>
          ) : (
            <Index
              chosen={current?.key ?? null}
              names={names}
              onChoose={choose}
              rows={rows}
            />
          )}
        </div>

        <div
          className={cn(
            "min-h-0 overflow-y-auto overscroll-contain px-5 py-5 sm:block sm:px-7 sm:py-6 [scrollbar-color:var(--v-rule)_transparent] [scrollbar-width:thin]",
            pane === "index" && "hidden",
          )}
        >
          <button
            className="mb-4 inline-flex min-h-11 items-center gap-2 font-ui text-meta text-mute outline-offset-3 hover:text-ink sm:hidden"
            onClick={() => setPane("index")}
            type="button"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            Back to the list
          </button>
          {current ? (
            <Panel key={current.key} labelledBy={`${names}-${current.key}`}>
              <div
                className={cn(
                  "flex min-w-0 flex-col gap-2.5",
                  current.off && "opacity-70",
                )}
              >
                {current.detail}
              </div>
            </Panel>
          ) : null}
        </div>
      </div>
    </div>
  );
}

// biome-ignore-start lint/a11y/noNoninteractiveTabindex: The panel scrolls by keyboard.
function Panel({
  children,
  labelledBy,
}: {
  children: ReactNode;
  labelledBy: string;
}) {
  return (
    <div
      aria-labelledby={labelledBy}
      className="min-w-0 outline-offset-8"
      role="tabpanel"
      tabIndex={0}
    >
      {children}
    </div>
  );
}
// biome-ignore-end lint/a11y/noNoninteractiveTabindex: End panel exception.

function Controls({
  includeOff,
  noun,
  off,
  onIncludeOff,
  onOrder,
  onSearch,
  order,
  search,
  shown,
  total,
}: {
  includeOff: boolean;
  noun: string;
  off: number;
  onIncludeOff: (includeOff: boolean) => void;
  onOrder: (order: CollectionOrder) => void;
  onSearch: (search: string) => void;
  order: CollectionOrder;
  search: string;
  shown: number;
  total: number;
}) {
  const searchField = useId();
  const orderField = useId();
  const box = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (window.matchMedia("(pointer: coarse)").matches) return;
    box.current?.focus();
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2.5">
      <div className="flex min-w-0 flex-[1_1_240px] items-center gap-2.5 rounded-control bg-deep px-3 text-mute sm:flex-[0_1_340px]">
        <Search aria-hidden="true" size={16} />
        <label className="sr-only" htmlFor={searchField}>
          Search the {noun}
        </label>
        <input
          className="min-h-11 min-w-0 flex-auto border-0 bg-transparent font-ui text-ui text-ink outline-offset-3"
          id={searchField}
          onChange={(event) => onSearch(event.target.value)}
          placeholder={`Search ${noun}`}
          ref={box}
          type="search"
          value={search}
        />
      </div>
      <div className="flex items-center gap-2">
        <label className="font-ui text-label text-mute" htmlFor={orderField}>
          Order
        </label>
        <select
          className="min-h-11 rounded-control bg-deep px-3 font-ui text-meta text-ink outline-offset-3"
          id={orderField}
          onChange={(event) => onOrder(event.target.value as CollectionOrder)}
          value={order}
        >
          <option value="given">As written</option>
          <option value="name">By name, A to Z</option>
        </select>
      </div>
      {off > 0 ? (
        <label className="inline-flex min-h-11 items-center gap-2 font-ui text-label text-mute">
          <input
            checked={includeOff}
            className="size-4 accent-[var(--v-action)]"
            onChange={(event) => onIncludeOff(event.target.checked)}
            type="checkbox"
          />
          Include the {off} that {off === 1 ? "is" : "are"} off
        </label>
      ) : null}
      {shown === total ? null : (
        <p className="font-ui text-label text-mute tabular-nums sm:ml-auto">
          {shown} of {total}
        </p>
      )}
    </div>
  );
}

function Index({
  chosen,
  names,
  onChoose,
  rows,
}: {
  chosen: string | null;
  names: string;
  onChoose: (key: string) => void;
  rows: Row[];
}) {
  const scroller = useRef<HTMLDivElement>(null);
  const wantsFocus = useRef(false);
  const virtual = useVirtualizer({
    count: rows.length,
    estimateSize: (index) =>
      rows[index].kind === "heading" ? HEADING_HEIGHT : ITEM_HEIGHT,
    getScrollElement: () => scroller.current,
    overscan: 8,
    useFlushSync: false,
  });

  const chosenRow = rows.findIndex(
    (row) => row.kind === "item" && row.item.key === chosen,
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: The chosen row is what asks to be scrolled to.
  useEffect(() => {
    if (chosenRow < 0) return;
    virtual.scrollToIndex(chosenRow, { align: "auto" });
  }, [chosenRow]);

  /** A row asked for by keyboard takes focus the moment the list mounts it. */
  function landFocus(button: HTMLButtonElement | null) {
    if (!button || !wantsFocus.current) return;
    wantsFocus.current = false;
    button.focus();
  }

  function moveFocus(event: KeyboardEvent<HTMLButtonElement>, from: number) {
    const to = nextItemRow(rows, event.key, from);
    if (to === null) return;
    event.preventDefault();
    const row = rows[to];
    if (row.kind !== "item") return;
    wantsFocus.current = true;
    onChoose(row.item.key);
  }

  return (
    <div
      aria-label="Items"
      aria-orientation="vertical"
      className="h-full max-h-full overflow-y-auto overscroll-contain px-2 py-2 [scrollbar-color:var(--v-rule)_transparent] [scrollbar-width:thin] sm:px-3 sm:py-3"
      ref={scroller}
      role="tablist"
    >
      <div
        className="relative w-full"
        style={{ height: virtual.getTotalSize() }}
      >
        {virtual.getVirtualItems().map((slot) => {
          const row = rows[slot.index];
          const place = {
            left: 0,
            position: "absolute" as const,
            top: 0,
            transform: `translateY(${slot.start}px)`,
            width: "100%",
          };
          if (row.kind === "heading") {
            return (
              <div
                className="flex items-end px-3 pb-1.5 font-ui text-label font-semibold tracking-[0.08em] text-mute uppercase"
                data-index={slot.index}
                key={row.key}
                ref={virtual.measureElement}
                style={{ ...place, minHeight: HEADING_HEIGHT }}
              >
                <span className="truncate">{row.group}</span>
              </div>
            );
          }
          const item = row.item;
          const selected = item.key === chosen;
          return (
            <div
              data-index={slot.index}
              key={row.key}
              ref={virtual.measureElement}
              style={place}
            >
              <button
                aria-selected={selected}
                className={cn(
                  "flex w-full cursor-pointer items-baseline gap-2.5 rounded-control px-3 py-2.5 text-left outline-offset-[-1px] hover:bg-deep/60 aria-selected:bg-deep aria-selected:shadow-[inset_2px_0_0_var(--v-action)]",
                  item.off && "opacity-60",
                )}
                data-row={item.key}
                id={`${names}-${item.key}`}
                onClick={() => onChoose(item.key)}
                onKeyDown={(event) => moveFocus(event, slot.index)}
                ref={selected ? landFocus : undefined}
                role="tab"
                tabIndex={selected ? 0 : -1}
                type="button"
              >
                <span
                  className="min-w-0 flex-1 truncate font-ui text-ui font-medium text-ink"
                  title={item.name}
                >
                  {item.name}
                </span>
                {item.note ? (
                  <span className="max-w-[45%] shrink-0 truncate font-ui text-label text-mute tabular-nums">
                    {item.note}
                  </span>
                ) : null}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function rowsOf(items: readonly CollectionItem[], grouped: boolean): Row[] {
  const rows: Row[] = [];
  let group: string | undefined;
  for (const item of items) {
    if (grouped && item.group && item.group !== group) {
      rows.push({
        group: item.group,
        key: `group:${item.group}`,
        kind: "heading",
      });
    }
    group = item.group;
    rows.push({ item, key: `item:${item.key}`, kind: "item" });
  }
  return rows;
}

function nextItemRow(rows: Row[], key: string, from: number): number | null {
  const items = rows
    .map((row, index) => (row.kind === "item" ? index : -1))
    .filter((index) => index >= 0);
  if (items.length === 0) return null;
  const at = items.indexOf(from);
  if (key === "ArrowDown") return items[Math.min(at + 1, items.length - 1)];
  if (key === "ArrowUp") return items[Math.max(at - 1, 0)];
  if (key === "Home") return items[0];
  if (key === "End") return items[items.length - 1];
  return null;
}
