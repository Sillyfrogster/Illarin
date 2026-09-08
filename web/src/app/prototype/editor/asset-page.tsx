"use client";

import { ArrowDown, ArrowUp, Eye, EyeOff, Plus, Search, X } from "lucide-react";
import Image from "next/image";
import { useEffect, useId, useRef, useState } from "react";
import darkArt from "@/assets/art/full/illarin-detail-page-art-dark-v2.webp";
import lightArt from "@/assets/art/full/illarin-detail-page-art-light-v2.webp";
import {
  BLOCK_WIDTHS,
  type BlockLayout,
  type BlockWidth,
  elementTracks,
  LAYOUT_LABELS,
  layoutChoiceIssue,
  NARROW_BLOCK_GRID_PX,
  packBlockRows,
  WIDTH_LABELS,
  widthChoiceIssue,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import {
  type Asset,
  type Block,
  blockRules,
  type Element,
  type Item,
} from "./data";
import { Inline } from "./inline";
import { cn, Eyebrow } from "./ui";

export const proseKey = (elementId: string) => `e:${elementId}`;
export const itemKey = (elementId: string, itemId: string, field: string) =>
  `i:${elementId}:${itemId}:${field}`;

function hasContent(element: Element) {
  return Boolean(
    element.text.trim() || element.items.some((item) => item.text.trim()),
  );
}

function newItem(entry: boolean): Item {
  return {
    id: `new-${Math.random().toString(36).slice(2, 8)}`,
    name: "",
    text: "",
    keys: entry ? "" : undefined,
    enabled: entry ? true : undefined,
  };
}

type Edit = {
  cursor: string | null;
  setCursor: (key: string | null) => void;
  live: boolean;
};

function ControlButton({
  children,
  label,
  onClick,
  disabled,
}: {
  children: React.ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      className="ws:inline-flex ws:size-8 ws:shrink-0 ws:items-center ws:justify-center ws:rounded-[10px] ws:text-mute ws:transition ws:hover:bg-ink/10 ws:hover:text-ink ws:disabled:opacity-30 ws:motion-reduce:transition-none ws:[&_svg]:size-4"
    >
      {children}
    </button>
  );
}

function Picker({
  value,
  onChange,
  label,
  children,
}: {
  value: string;
  onChange: (value: string) => void;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <select
      aria-label={label}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="ws:min-h-8 ws:appearance-none ws:rounded-[10px] ws:border-0 ws:bg-transparent ws:px-2.5 ws:text-xs ws:font-semibold ws:text-mute ws:outline-none ws:hover:bg-ink/10 ws:hover:text-ink"
    >
      {children}
    </select>
  );
}

function BlockControls({
  block,
  index,
  total,
  narrow,
  update,
  move,
  note,
}: {
  block: Block;
  index: number;
  total: number;
  narrow: boolean;
  update: (change: Partial<Block>) => void;
  move: (delta: number) => void;
  note: (message: string) => void;
}) {
  const rules = blockRules[block.definition];
  return (
    <div className="w-tools ws:shrink-0 ws:opacity-0 ws:transition ws:group-focus-within/block:opacity-100 ws:group-hover/block:opacity-100 ws:motion-reduce:transition-none">
      {!narrow && (
        <Picker
          label={`${block.title} width`}
          value={block.width}
          onChange={(next) => {
            const reason = widthChoiceIssue(block.layout, next as BlockWidth);
            if (reason) note(reason);
            else update({ width: next as BlockWidth });
          }}
        >
          {BLOCK_WIDTHS.map((width) => (
            <option key={width} value={width}>
              {WIDTH_LABELS[width]}
            </option>
          ))}
        </Picker>
      )}
      {rules.layouts.length > 1 && (
        <Picker
          label={`${block.title} layout`}
          value={block.layout}
          onChange={(next) => {
            const reason = layoutChoiceIssue(
              next as BlockLayout,
              block.width,
              block.elements.map((element) => element.label),
            );
            if (reason) note(reason);
            else update({ layout: next as BlockLayout });
          }}
        >
          {rules.layouts.map((layout) => (
            <option key={layout} value={layout}>
              {LAYOUT_LABELS[layout]}
            </option>
          ))}
        </Picker>
      )}
      <ControlButton
        label={`Move ${block.title} earlier`}
        disabled={index === 0}
        onClick={() => move(-1)}
      >
        <ArrowUp />
      </ControlButton>
      <ControlButton
        label={`Move ${block.title} later`}
        disabled={index === total - 1}
        onClick={() => move(1)}
      >
        <ArrowDown />
      </ControlButton>
      {rules.hideable && (
        <ControlButton
          label={block.hidden ? `Show ${block.title}` : `Hide ${block.title}`}
          onClick={() => update({ hidden: !block.hidden })}
        >
          {block.hidden ? <Eye /> : <EyeOff />}
        </ControlButton>
      )}
    </div>
  );
}

function ItemRow({
  item,
  element,
  edit,
  update,
  remove,
}: {
  item: Item;
  element: Element;
  edit: Edit;
  update: (item: Item) => void;
  remove: () => void;
}) {
  const nameKey = itemKey(element.id, item.id, "name");
  const textKey = itemKey(element.id, item.id, "text");
  return (
    <li className="ws:group/item ws:relative ws:min-w-0">
      <div className="ws:flex ws:min-w-0 ws:items-baseline ws:gap-3">
        <Inline
          as="span"
          label={`${element.label} name`}
          value={item.name}
          singleLine
          live={edit.live}
          active={edit.cursor === nameKey}
          activate={() => edit.setCursor(nameKey)}
          done={() => edit.setCursor(null)}
          onChange={(name) => update({ ...item, name })}
          placeholder="Name this one"
          className="ws:min-w-0 ws:flex-1 ws:font-semibold ws:tracking-tight"
        />
        {edit.live && (
          <span className="ws:flex ws:shrink-0 ws:items-center ws:opacity-0 ws:transition ws:group-focus-within/item:opacity-100 ws:group-hover/item:opacity-100 ws:motion-reduce:transition-none">
            <ControlButton
              label={`Remove ${item.name || "this one"}`}
              onClick={remove}
            >
              <X />
            </ControlButton>
          </span>
        )}
      </div>
      <Inline
        label={`${item.name || "Untitled"} text`}
        value={item.text}
        live={edit.live}
        active={edit.cursor === textKey}
        activate={() => edit.setCursor(textKey)}
        done={() => edit.setCursor(null)}
        onChange={(text) => update({ ...item, text })}
        placeholder="Write this one."
        className={cn(
          "w-writing ws:mt-2 ws:max-w-[62ch] ws:text-[1.0625rem] ws:leading-7 ws:text-mute",
        )}
      />
    </li>
  );
}

/** A book reads as its index beside the one entry that is open */
function EntryTable({
  element,
  edit,
  update,
}: {
  element: Element;
  edit: Edit;
  update: (element: Element) => void;
}) {
  const [search, setSearch] = useState("");
  const [picked, setPicked] = useState<string | null>(null);
  const list = useRef<HTMLDivElement>(null);
  const names = useId();

  const wanted = search.trim().toLowerCase();
  const shownItems = element.items.filter((item) => {
    if (!edit.live && !item.text.trim()) return false;
    if (!wanted) return true;
    return `${item.name} ${item.keys ?? ""}`.toLowerCase().includes(wanted);
  });
  const fromCursor = edit.cursor?.startsWith(`i:${element.id}:`)
    ? edit.cursor.split(":")[2]
    : undefined;
  const open =
    element.items.find((item) => item.id === (fromCursor ?? picked)) ??
    shownItems[0];
  const row = shownItems.findIndex((item) => item.id === open?.id);

  useEffect(() => {
    if (row < 0) return;
    list.current?.children[row]?.scrollIntoView({ block: "nearest" });
  }, [row]);

  function change(next: Item) {
    update({
      ...element,
      items: element.items.map((one) => (one.id === next.id ? next : one)),
    });
  }

  return (
    <div className="w-book ws:min-w-0">
      <div className="ws:min-w-0">
        <div className="ws:flex ws:min-h-11 ws:items-center ws:gap-2.5 ws:px-1 ws:shadow-[inset_0_-1px_0_var(--w-line)]">
          <Search className="ws:size-4 ws:shrink-0 ws:text-mute" />
          <input
            type="search"
            value={search}
            aria-label={`Search ${element.label.toLowerCase()} by name and key`}
            placeholder="Search names and keys"
            onChange={(event) => setSearch(event.target.value)}
            className="ws:min-w-0 ws:flex-1 ws:border-0 ws:bg-transparent ws:p-0 ws:text-sm ws:shadow-none ws:outline-none ws:placeholder:text-mute"
          />
          <span className="ws:shrink-0 ws:text-xs ws:text-mute ws:tabular-nums">
            {shownItems.length === element.items.length
              ? element.items.length
              : `${shownItems.length} of ${element.items.length}`}
          </span>
        </div>
        <div
          ref={list}
          role="tablist"
          aria-label={`The entries in ${element.label.toLowerCase()}`}
          aria-orientation="vertical"
          className="w-index"
        >
          {shownItems.map((item, index) => (
            <button
              key={item.id}
              type="button"
              role="tab"
              id={`${names}-${item.id}`}
              aria-selected={item.id === open?.id}
              tabIndex={item.id === open?.id ? 0 : -1}
              data-off={item.enabled === false ? true : undefined}
              onClick={() => {
                setPicked(item.id);
                edit.setCursor(null);
              }}
              onKeyDown={(event) => {
                const to =
                  event.key === "ArrowDown"
                    ? index + 1
                    : event.key === "ArrowUp"
                      ? index - 1
                      : event.key === "Home"
                        ? 0
                        : event.key === "End"
                          ? shownItems.length - 1
                          : null;
                if (to === null || to < 0 || to >= shownItems.length) return;
                event.preventDefault();
                setPicked(shownItems[to].id);
                edit.setCursor(null);
                list.current?.querySelectorAll("button")[to]?.focus();
              }}
            >
              <span className="ws:min-w-0 ws:truncate ws:font-semibold">
                {item.name || "Unnamed"}
              </span>
              <span className="ws:min-w-0 ws:truncate ws:text-xs ws:text-mute">
                {item.enabled === false ? "Off" : item.keys || "No keys"}
              </span>
            </button>
          ))}
        </div>
        {edit.live && (
          <button
            type="button"
            onClick={() => {
              const created = newItem(true);
              update({ ...element, items: [...element.items, created] });
              setSearch("");
              setPicked(created.id);
              edit.setCursor(itemKey(element.id, created.id, "name"));
            }}
            className="ws:mt-3 ws:inline-flex ws:min-h-9 ws:items-center ws:gap-2 ws:rounded-full ws:px-3 ws:text-sm ws:font-semibold ws:text-mute ws:transition ws:hover:bg-ink/10 ws:hover:text-ink ws:motion-reduce:transition-none ws:[&_svg]:size-4"
          >
            <Plus />
            Add an entry
          </button>
        )}
      </div>

      {open && (
        <div
          role="tabpanel"
          aria-labelledby={`${names}-${open.id}`}
          className="ws:group/item ws:min-w-0"
        >
          <div className="ws:flex ws:min-w-0 ws:items-baseline ws:gap-3">
            <Inline
              as="h4"
              label="Entry name"
              value={open.name}
              singleLine
              live={edit.live}
              active={edit.cursor === itemKey(element.id, open.id, "name")}
              activate={() =>
                edit.setCursor(itemKey(element.id, open.id, "name"))
              }
              done={() => edit.setCursor(null)}
              onChange={(name) => change({ ...open, name })}
              placeholder="Name this entry"
              className="ws:min-w-0 ws:flex-1 ws:font-display ws:text-2xl ws:font-medium"
            />
            {edit.live && (
              <span className="ws:flex ws:shrink-0 ws:items-center ws:opacity-0 ws:transition ws:group-focus-within/item:opacity-100 ws:group-hover/item:opacity-100 ws:motion-reduce:transition-none">
                <ControlButton
                  label={
                    open.enabled === false
                      ? `Enable ${open.name || "entry"}`
                      : `Disable ${open.name || "entry"}`
                  }
                  onClick={() =>
                    change({ ...open, enabled: open.enabled === false })
                  }
                >
                  {open.enabled === false ? <EyeOff /> : <Eye />}
                </ControlButton>
                <ControlButton
                  label={`Remove ${open.name || "this entry"}`}
                  onClick={() => {
                    setPicked(null);
                    edit.setCursor(null);
                    update({
                      ...element,
                      items: element.items.filter((one) => one.id !== open.id),
                    });
                  }}
                >
                  <X />
                </ControlButton>
              </span>
            )}
          </div>
          {(edit.live || open.keys) && (
            <Inline
              as="p"
              label={`${open.name || "Entry"} keys`}
              value={open.keys ?? ""}
              singleLine
              live={edit.live}
              active={edit.cursor === itemKey(element.id, open.id, "keys")}
              activate={() =>
                edit.setCursor(itemKey(element.id, open.id, "keys"))
              }
              done={() => edit.setCursor(null)}
              onChange={(keys) => change({ ...open, keys })}
              placeholder="Keys that call this entry"
              className="ws:mt-2 ws:text-sm ws:text-mute"
            />
          )}
          <Inline
            label={`${open.name || "Entry"} text`}
            value={open.text}
            live={edit.live}
            active={edit.cursor === itemKey(element.id, open.id, "text")}
            activate={() =>
              edit.setCursor(itemKey(element.id, open.id, "text"))
            }
            done={() => edit.setCursor(null)}
            onChange={(text) => change({ ...open, text })}
            placeholder="Write this entry."
            className="w-writing ws:mt-5 ws:max-w-[62ch] ws:text-[1.0625rem] ws:leading-7"
          />
        </div>
      )}
    </div>
  );
}

function ElementBody({
  element,
  edit,
  update,
}: {
  element: Element;
  edit: Edit;
  update: (element: Element) => void;
}) {
  if (element.type === "prose")
    return (
      <Inline
        label={element.label}
        value={element.text}
        live={edit.live}
        active={edit.cursor === proseKey(element.id)}
        activate={() => edit.setCursor(proseKey(element.id))}
        done={() => edit.setCursor(null)}
        onChange={(text) => update({ ...element, text })}
        placeholder={`Write the ${element.label.toLowerCase()}.`}
        className="w-writing ws:max-w-[64ch]"
      />
    );

  if (element.type === "entry_table")
    return <EntryTable element={element} edit={edit} update={update} />;

  const conversation = element.type === "dialogue_sample";
  const items = edit.live
    ? element.items
    : element.items.filter((item) => item.text.trim());
  return (
    <div className="ws:min-w-0">
      <ul className={cn("ws:min-w-0", !conversation && "w-items")}>
        {items.map((item) => (
          <ItemRow
            key={item.id}
            item={item}
            element={element}
            edit={edit}
            update={(next) =>
              update({
                ...element,
                items: element.items.map((one) =>
                  one.id === next.id ? next : one,
                ),
              })
            }
            remove={() =>
              update({
                ...element,
                items: element.items.filter((one) => one.id !== item.id),
              })
            }
          />
        ))}
      </ul>
      {edit.live && (
        <button
          type="button"
          onClick={() => {
            const created = newItem(false);
            update({ ...element, items: [...element.items, created] });
            edit.setCursor(itemKey(element.id, created.id, "name"));
          }}
          className="ws:mt-6 ws:inline-flex ws:min-h-9 ws:items-center ws:gap-2 ws:rounded-full ws:px-3 ws:text-sm ws:font-semibold ws:text-mute ws:transition ws:hover:bg-ink/10 ws:hover:text-ink ws:motion-reduce:transition-none ws:[&_svg]:size-4"
        >
          <Plus />
          Add to {element.label.toLowerCase()}
        </button>
      )}
    </div>
  );
}

function BlockSection({
  block,
  index,
  total,
  edit,
  changed,
  width,
  update,
  move,
  note,
}: {
  block: Block;
  index: number;
  total: number;
  edit: Edit;
  changed: Set<string>;
  width?: number;
  update: (change: Partial<Block>) => void;
  move: (delta: number) => void;
  note: (message: string) => void;
}) {
  const elements = edit.live
    ? block.elements
    : block.elements.filter(hasContent);
  if (!elements.length) return null;
  const narrow = width !== undefined && width <= NARROW_BLOCK_GRID_PX;
  return (
    <section
      className={cn(
        "ws:group/block ws:relative ws:min-w-0",
        edit.live && "w-block",
        edit.live && block.hidden && "ws:opacity-55",
      )}
    >
      <header className="ws:mb-7 ws:flex ws:min-h-9 ws:items-center ws:justify-between ws:gap-4">
        <h2 className="ws:min-w-0 ws:font-display ws:text-[1.75rem] ws:leading-tight ws:font-medium ws:wrap-anywhere ws:md:text-[2.125rem]">
          {block.title || "Untitled block"}
        </h2>
        {edit.live && (
          <BlockControls
            block={block}
            index={index}
            total={total}
            narrow={narrow}
            update={update}
            move={move}
            note={note}
          />
        )}
      </header>
      <div
        className="ws:grid ws:gap-x-9 ws:gap-y-12"
        style={{
          gridTemplateColumns: narrow
            ? "minmax(0,1fr)"
            : elementTracks(block.layout, elements.length),
        }}
      >
        {elements.map((element) => (
          <div key={element.id} className="w-element ws:min-w-0">
            {(elements.length > 1 || element.type !== "prose") && (
              <Eyebrow
                className={cn(
                  "ws:mb-3 ws:flex ws:items-center ws:gap-2",
                  changed.has(element.id) && "ws:text-amber",
                )}
              >
                {changed.has(element.id) && (
                  <span className="ws:size-1.5 ws:rounded-full ws:bg-amber" />
                )}
                {element.label}
              </Eyebrow>
            )}
            <ElementBody
              element={element}
              edit={edit}
              update={(next) =>
                update({
                  elements: block.elements.map((one) =>
                    one.id === next.id ? next : one,
                  ),
                })
              }
            />
          </div>
        ))}
      </div>
    </section>
  );
}

export function AssetPage({
  asset,
  live,
  cursor,
  setCursor,
  theme,
  changed,
  update,
  note,
  action,
}: {
  asset: Asset;
  live: boolean;
  cursor: string | null;
  setCursor: (key: string | null) => void;
  theme: "light" | "dark";
  changed: Set<string>;
  update: (asset: Asset) => void;
  note: (message: string) => void;
  action: React.ReactNode;
}) {
  const [ref, width] = useMeasuredWidth<HTMLDivElement>();
  const edit: Edit = { cursor, setCursor, live };
  const visible = live ? asset.blocks : asset.blocks.filter((b) => !b.hidden);
  const rows = packBlockRows(visible, { availableWidth: width });

  function patchBlock(id: string, change: Partial<Block>) {
    update({
      ...asset,
      blocks: asset.blocks.map((block) =>
        block.id === id ? { ...block, ...change } : block,
      ),
    });
  }
  function moveBlock(id: string, delta: number) {
    const from = asset.blocks.findIndex((block) => block.id === id);
    const to = from + delta;
    if (to < 0 || to >= asset.blocks.length) return;
    const blocks = [...asset.blocks];
    [blocks[from], blocks[to]] = [blocks[to], blocks[from]];
    update({ ...asset, blocks });
  }

  return (
    <div className="ws:w-full ws:overflow-x-clip">
      <div className="ws:relative ws:mx-auto ws:max-w-[86rem] ws:px-5 ws:pt-10 ws:pb-20 ws:md:px-10 ws:md:pt-16 ws:md:pb-28">
        <Image
          src={theme === "light" ? lightArt : darkArt}
          alt=""
          priority
          sizes="(max-width: 900px) 100vw, 900px"
          style={{
            maskImage:
              "linear-gradient(to left, black 6%, transparent 82%), linear-gradient(to top, transparent 2%, black 34%)",
            maskComposite: "intersect",
            WebkitMaskComposite: "source-in",
          }}
          className="ws:pointer-events-none ws:absolute ws:top-0 ws:right-[-10%] ws:h-full ws:w-[72%] ws:object-cover ws:opacity-50 ws:max-md:opacity-20"
        />
        <div
          className={cn(
            "ws:group/block ws:relative ws:max-w-3xl",
            live && "w-block",
          )}
        >
          <Eyebrow className={cn(changed.has("name") && "ws:text-amber")}>
            {asset.kind} · {asset.version || "No version label"}
            {asset.nsfw === "yes" && " · Adult content"}
          </Eyebrow>
          <Inline
            as="h1"
            label="Asset name"
            value={asset.name}
            singleLine
            live={live}
            active={cursor === "name"}
            activate={() => setCursor("name")}
            done={() => setCursor(null)}
            onChange={(name) => update({ ...asset, name })}
            placeholder="Name this asset"
            className="ws:mt-5 ws:font-display ws:text-[clamp(2.75rem,7vw,5.25rem)] ws:leading-[1.03] ws:font-medium ws:tracking-[-0.015em]"
          />
          <Inline
            label="Blurb"
            value={asset.blurb}
            live={live}
            active={cursor === "blurb"}
            activate={() => setCursor("blurb")}
            done={() => setCursor(null)}
            onChange={(blurb) => update({ ...asset, blurb })}
            placeholder="Add the line that sits under the title."
            className="w-writing ws:mt-6 ws:max-w-2xl ws:text-[1.3125rem] ws:leading-9 ws:text-mute"
          />
          <div className="ws:mt-10 ws:flex ws:flex-wrap ws:items-center ws:gap-3">
            {action}
          </div>
        </div>
      </div>

      <div
        ref={ref}
        className="ws:mx-auto ws:flex ws:max-w-[86rem] ws:flex-col ws:gap-[88px] ws:px-5 ws:pb-44 ws:md:px-10"
      >
        {rows.map((row) => (
          <div
            key={row[0].block.id}
            className="ws:-mx-4 ws:grid ws:grid-cols-12 ws:items-start ws:gap-y-[88px]"
          >
            {row.map(({ block, columns, startColumn }) => (
              <div
                key={block.id}
                style={{ gridColumn: `${startColumn} / span ${columns}` }}
                className="ws:min-w-0 ws:px-4"
              >
                <BlockSection
                  block={block}
                  index={asset.blocks.findIndex((one) => one.id === block.id)}
                  total={asset.blocks.length}
                  edit={edit}
                  changed={changed}
                  width={width}
                  update={(change) => patchBlock(block.id, change)}
                  move={(delta) => moveBlock(block.id, delta)}
                  note={note}
                />
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
