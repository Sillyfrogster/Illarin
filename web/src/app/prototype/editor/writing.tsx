"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ChevronDown, ChevronUp, Plus, Search, Trash2, X } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { DETAILS_REGION, elementRegion } from "./asset-page";
import type { Asset, Element, Item } from "./data";
import type { Focus } from "./session";
import { Button, cn, Field, Input, Select, Textarea, WritingArea } from "./ui";

const RISE = { duration: 0.42, ease: [0.22, 1, 0.36, 1] as const };

function newItem(prefix: string): Item {
  return {
    id: `${prefix}-${Math.random().toString(36).slice(2, 8)}`,
    name: "",
    text: "",
    keys: prefix === "entry" ? "" : undefined,
    enabled: prefix === "entry" ? true : undefined,
  };
}

function DetailsForm({
  asset,
  update,
  readOnly,
}: {
  asset: Asset;
  update: (asset: Asset) => void;
  readOnly: boolean;
}) {
  return (
    <fieldset disabled={readOnly} className="ws:space-y-7">
      <Field label="Name">
        <Input
          data-writing-input
          value={asset.name}
          onChange={(e) => update({ ...asset, name: e.target.value })}
          className="ws:min-h-14 ws:font-display ws:text-2xl"
        />
      </Field>
      <Field
        label="Blurb"
        hint="The short line under the title on the public page."
      >
        <Textarea
          rows={3}
          value={asset.blurb}
          onChange={(e) => update({ ...asset, blurb: e.target.value })}
        />
      </Field>
      <div className="ws:grid ws:gap-6 ws:sm:grid-cols-2">
        <Field label="Version label" hint="Your own name for this version.">
          <Input
            value={asset.version}
            onChange={(e) => update({ ...asset, version: e.target.value })}
          />
        </Field>
        <Field label="Adult content">
          <Select
            value={asset.nsfw}
            onChange={(e) =>
              update({ ...asset, nsfw: e.target.value as Asset["nsfw"] })
            }
          >
            <option value="unanswered">Not answered yet</option>
            <option value="no">No adult content</option>
            <option value="yes">Contains adult content</option>
          </Select>
        </Field>
      </div>
    </fieldset>
  );
}

function ItemRail({
  element,
  itemId,
  select,
  readOnly,
  setItems,
}: {
  element: Element;
  itemId: string;
  select: (id: string) => void;
  readOnly: boolean;
  setItems: (items: Item[]) => void;
}) {
  const [query, setQuery] = useState("");
  const searchable = element.items.length > 8;
  const matches = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return element.items;
    return element.items.filter((item) =>
      `${item.name} ${item.keys ?? ""} ${item.text}`
        .toLowerCase()
        .includes(needle),
    );
  }, [element.items, query]);
  const prefix = element.type === "entry_table" ? "entry" : "item";
  return (
    <div className="ws:flex ws:min-h-0 ws:flex-col ws:gap-3 ws:lg:h-full">
      {searchable && (
        <div className="ws:relative ws:shrink-0">
          <Search className="ws:pointer-events-none ws:absolute ws:top-1/2 ws:left-3.5 ws:size-4 ws:-translate-y-1/2 ws:text-mute" />
          <Input
            type="search"
            value={query}
            placeholder={`Filter ${element.items.length} entries`}
            aria-label={`Filter ${element.label}`}
            onChange={(e) => setQuery(e.target.value)}
            className="ws:pl-10 ws:text-sm"
          />
        </div>
      )}
      <ul className="ws:-mx-1.5 ws:min-h-0 ws:flex-1 ws:overflow-y-auto ws:overscroll-contain ws:px-1.5 ws:max-h-56 ws:lg:max-h-[52dvh]">
        {matches.map((item) => (
          <li key={item.id}>
            <button
              type="button"
              onClick={() => select(item.id)}
              aria-current={item.id === itemId ? "true" : undefined}
              className={cn(
                "ws:flex ws:min-h-11 ws:w-full ws:items-center ws:gap-2.5 ws:rounded-lg ws:px-3 ws:py-2 ws:text-left ws:text-sm ws:transition ws:hover:bg-ink/6 ws:motion-reduce:transition-none",
                item.id === itemId && "ws:bg-ink/10 ws:font-semibold",
              )}
            >
              <span className="ws:min-w-0 ws:flex-1 ws:truncate">
                {item.name || "Untitled"}
              </span>
              {!item.text.trim() && (
                <span className="ws:size-1.5 ws:shrink-0 ws:rounded-full ws:bg-amber" />
              )}
            </button>
          </li>
        ))}
        {!matches.length && (
          <li className="ws:px-3 ws:py-4 ws:text-sm ws:text-mute">
            Nothing matches “{query}”.
          </li>
        )}
      </ul>
      <Button
        size="small"
        variant="outline"
        disabled={readOnly}
        className="ws:shrink-0 ws:self-start"
        onClick={() => {
          const created = newItem(prefix);
          setItems([...element.items, created]);
          select(created.id);
        }}
      >
        <Plus />
        Add
      </Button>
    </div>
  );
}

function ItemForm({
  element,
  item,
  update,
  move,
  remove,
  readOnly,
  index,
  total,
}: {
  element: Element;
  item: Item;
  update: (item: Item) => void;
  move: (delta: number) => void;
  remove: () => void;
  readOnly: boolean;
  index: number;
  total: number;
}) {
  const entry = element.type === "entry_table";
  return (
    <fieldset disabled={readOnly} className="ws:min-w-0 ws:space-y-6">
      <div className="ws:grid ws:gap-5 ws:sm:grid-cols-2">
        <Field label={element.type === "dialogue_sample" ? "Speaker" : "Name"}>
          <Input
            value={item.name}
            onChange={(e) => update({ ...item, name: e.target.value })}
          />
        </Field>
        {entry && (
          <Field label="Keys" hint="Comma separated.">
            <Input
              value={item.keys ?? ""}
              onChange={(e) => update({ ...item, keys: e.target.value })}
            />
          </Field>
        )}
      </div>
      <WritingArea
        data-writing-input
        aria-label={`${item.name || "Untitled"} text`}
        value={item.text}
        rows={10}
        placeholder="Write this one."
        onChange={(e) => update({ ...item, text: e.target.value })}
        className="ws:min-h-64 ws:max-w-none"
      />
      <div className="ws:flex ws:flex-wrap ws:items-center ws:gap-2">
        {entry && (
          <Button
            size="small"
            variant="outline"
            onClick={() => update({ ...item, enabled: item.enabled === false })}
          >
            {item.enabled === false ? "Disabled" : "Enabled"}
          </Button>
        )}
        <Button
          size="small"
          aria-label="Move earlier"
          disabled={index === 0}
          onClick={() => move(-1)}
        >
          <ChevronUp />
        </Button>
        <Button
          size="small"
          aria-label="Move later"
          disabled={index === total - 1}
          onClick={() => move(1)}
        >
          <ChevronDown />
        </Button>
        <span className="ws:text-xs ws:text-mute">
          {index + 1} of {total}
        </span>
        <Button
          size="small"
          className="ws:ml-auto ws:text-mute ws:hover:bg-critical-field ws:hover:text-critical"
          onClick={remove}
        >
          <Trash2 />
          Remove
        </Button>
      </div>
    </fieldset>
  );
}

/** The focused region, lifted off the page to write in and dropped back afterwards */
export function WritingSurface({
  asset,
  focus,
  update,
  close,
  selectItem,
  readOnly,
}: {
  asset: Asset;
  focus: Focus;
  update: (asset: Asset) => void;
  close: () => void;
  selectItem: (itemId: string) => void;
  readOnly: boolean;
}) {
  const reduced = useReducedMotion();
  const panel = useRef<HTMLDivElement>(null);
  const carets = useRef(new Map<string, [number, number]>());

  const block =
    focus?.type === "element"
      ? asset.blocks.find((b) => b.id === focus.blockId)
      : undefined;
  const element =
    focus?.type === "element"
      ? block?.elements.find((e) => e.id === focus.elementId)
      : undefined;
  const item =
    element && focus?.type === "element"
      ? (element.items.find((i) => i.id === focus.itemId) ?? element.items[0])
      : undefined;
  const caretKey = `${element?.id ?? "details"}:${item?.id ?? "text"}`;

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.stopPropagation();
        close();
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [close]);

  useLayoutEffect(() => {
    const input = panel.current?.querySelector<HTMLTextAreaElement>(
      "[data-writing-input]",
    );
    if (!input) return;
    input.focus({ preventScroll: true });
    const caret = carets.current.get(caretKey);
    if (caret) input.setSelectionRange(caret[0], caret[1]);
    else input.setSelectionRange(input.value.length, input.value.length);
  }, [caretKey]);

  function rememberCaret() {
    const input = panel.current?.querySelector<HTMLTextAreaElement>(
      "[data-writing-input]",
    );
    if (input)
      carets.current.set(caretKey, [
        input.selectionStart ?? 0,
        input.selectionEnd ?? 0,
      ]);
  }

  function updateElement(next: Element) {
    if (!block) return;
    update({
      ...asset,
      blocks: asset.blocks.map((b) =>
        b.id === block.id
          ? {
              ...b,
              elements: b.elements.map((e) => (e.id === next.id ? next : e)),
            }
          : b,
      ),
    });
  }

  if (!focus) return null;
  const id =
    focus.type === "details" ? DETAILS_REGION : elementRegion(focus.elementId);
  const heading =
    focus.type === "details" ? "Asset details" : (element?.label ?? "Content");
  const context =
    focus.type === "details"
      ? "Shown at the top of the page"
      : (block?.title ?? "");
  const collection = element && element.type !== "prose";

  return (
    <>
      <motion.button
        type="button"
        aria-label="Stop writing this and return to the page"
        onClick={close}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: reduced ? 0 : 0.3 }}
        className="ws:fixed ws:inset-0 ws:z-40 ws:cursor-default ws:bg-paper/70 ws:backdrop-blur-[6px]"
      />
      <motion.div
        layoutId={id}
        ref={panel}
        role="dialog"
        aria-label={`Writing ${heading}`}
        transition={reduced ? { duration: 0 } : RISE}
        onBlurCapture={rememberCaret}
        className={cn(
          "ws:fixed ws:inset-x-0 ws:top-[max(2.5dvh,1rem)] ws:z-50 ws:mx-auto ws:flex ws:max-h-[84dvh] ws:flex-col ws:overflow-hidden ws:rounded-[24px] ws:bg-card ws:text-ink ws:shadow-[var(--w-lift)]",
          collection ? "ws:w-[min(76rem,94vw)]" : "ws:w-[min(56rem,94vw)]",
        )}
      >
        <div aria-hidden="true" className="w-spectral-rule ws:shrink-0" />
        <header
          className={cn(
            "ws:flex ws:w-full ws:shrink-0 ws:items-start ws:justify-between ws:gap-5 ws:px-5 ws:pt-5 ws:pb-4 ws:md:px-9 ws:md:pt-7",
            !collection && "ws:mx-auto ws:max-w-[calc(68ch+4.5rem)]",
          )}
        >
          <div className="ws:min-w-0">
            {context && (
              <p className="ws:text-[0.6875rem] ws:font-bold ws:uppercase ws:tracking-[0.18em] ws:text-mute">
                {context}
              </p>
            )}
            <h2 className="ws:mt-1.5 ws:font-display ws:text-3xl ws:leading-tight ws:font-medium ws:wrap-anywhere ws:md:text-[2.5rem]">
              {heading}
            </h2>
          </div>
          <Button size="icon" onClick={close} aria-label="Back to the page">
            <X />
          </Button>
        </header>
        <div
          className={cn(
            "ws:min-h-0 ws:w-full ws:flex-1 ws:overflow-y-auto ws:overscroll-contain ws:px-5 ws:pb-7 ws:md:px-9",
            !collection && "ws:mx-auto ws:max-w-[calc(68ch+4.5rem)]",
          )}
        >
          {focus.type === "details" && (
            <DetailsForm asset={asset} update={update} readOnly={readOnly} />
          )}
          {element?.type === "prose" && (
            <WritingArea
              data-writing-input
              aria-label={element.label}
              value={element.text}
              rows={14}
              disabled={readOnly}
              placeholder={`Write the ${element.label.toLowerCase()}.`}
              onChange={(e) =>
                updateElement({ ...element, text: e.target.value })
              }
              className="ws:min-h-[46dvh]"
            />
          )}
          {collection && element && (
            <div className="ws:grid ws:min-h-0 ws:gap-7 ws:lg:grid-cols-[16rem_minmax(0,1fr)]">
              <ItemRail
                element={element}
                itemId={item?.id ?? ""}
                select={selectItem}
                readOnly={readOnly}
                setItems={(items) => updateElement({ ...element, items })}
              />
              {item ? (
                <ItemForm
                  element={element}
                  item={item}
                  readOnly={readOnly}
                  index={element.items.findIndex((i) => i.id === item.id)}
                  total={element.items.length}
                  update={(next) =>
                    updateElement({
                      ...element,
                      items: element.items.map((i) =>
                        i.id === next.id ? next : i,
                      ),
                    })
                  }
                  move={(delta) => {
                    const from = element.items.findIndex(
                      (i) => i.id === item.id,
                    );
                    const to = from + delta;
                    if (to < 0 || to >= element.items.length) return;
                    const items = [...element.items];
                    const [moved] = items.splice(from, 1);
                    items.splice(to, 0, moved);
                    updateElement({ ...element, items });
                  }}
                  remove={() => {
                    const items = element.items.filter((i) => i.id !== item.id);
                    updateElement({ ...element, items });
                    if (items.length) selectItem(items[0].id);
                  }}
                />
              ) : (
                <p className="ws:self-start ws:text-sm ws:text-mute">
                  Nothing here yet. Add the first one.
                </p>
              )}
            </div>
          )}
        </div>
        <footer className="ws:flex ws:w-full ws:shrink-0 ws:items-center ws:justify-between ws:gap-4 ws:px-5 ws:py-3.5 ws:text-xs ws:text-mute ws:shadow-[inset_0_1px_0_var(--w-hairline)] ws:md:px-9">
          <span>
            {focus.type === "details"
              ? "Saved with your private working copy"
              : `${(item?.text ?? element?.text ?? "").length.toLocaleString()} characters`}
          </span>
          <span className="ws:hidden ws:sm:inline">
            Esc returns to the page
          </span>
        </footer>
      </motion.div>
    </>
  );
}

export function WritingLayer(props: Parameters<typeof WritingSurface>[0]) {
  return (
    <AnimatePresence>
      {props.focus && <WritingSurface {...props} />}
    </AnimatePresence>
  );
}
