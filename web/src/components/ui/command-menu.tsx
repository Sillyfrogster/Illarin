"use client";

import { ArrowDown, ArrowUp, CornerDownLeft, Search } from "lucide-react";
import {
  type ComponentType,
  type KeyboardEvent,
  type ReactNode,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { Kbd } from "@/components/ui/kbd";
import { cn } from "@/lib/cn";
import { useFluidHover } from "@/lib/use-fluid-hover";

type CommandItem = {
  value: string;
  label: ReactNode;
  description?: ReactNode;
  icon?: ComponentType<{ className?: string; "aria-hidden"?: boolean }>;
  group?: string;
  trailing?: ReactNode;
  disabled?: boolean;
};

/** CommandMenu is Fluid Functionalism's command menu: type to narrow, arrows and Enter to choose, with one plate that follows the keys and the pointer. The caller filters the items. */
export function CommandMenu({
  items,
  query,
  onQueryChange,
  onSelect,
  label,
  placeholder,
  empty,
  autoFocus,
  className,
  listClassName,
}: {
  items: readonly CommandItem[];
  query: string;
  onQueryChange: (query: string) => void;
  onSelect: (item: CommandItem) => void;
  label: string;
  placeholder: string;
  empty: ReactNode;
  autoFocus?: boolean;
  className?: string;
  listClassName?: string;
}) {
  const listId = useId();
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef);
  const { registerItem, itemRects, isMeasured, remeasure } = hover;
  const firstEnabled = items.findIndex((item) => !item.disabled);
  const [active, setActive] = useState(firstEnabled);
  const count = items.length;
  const itemRefs = useMemo(
    () =>
      Array.from(
        { length: count },
        (_, index) => (element: HTMLElement | null) =>
          registerItem(index, element),
      ),
    [count, registerItem],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: a new query is what moves the highlight back to the top
  useEffect(() => {
    setActive(firstEnabled);
    remeasure();
  }, [query, firstEnabled, remeasure]);

  function move(step: 1 | -1) {
    if (firstEnabled < 0) return;
    let next = active;
    do {
      next = (next + step + count) % count;
    } while (items[next].disabled && next !== active);
    setActive(next);
    document
      .getElementById(`${listId}-${next}`)
      ?.scrollIntoView({ block: "nearest" });
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      move(event.key === "ArrowDown" ? 1 : -1);
    } else if (
      event.key === "Enter" &&
      items[active] &&
      !items[active].disabled
    ) {
      event.preventDefault();
      onSelect(items[active]);
    }
  }

  const rect = isMeasured && active >= 0 ? itemRects[active] : undefined;
  let group: string | undefined;

  return (
    <div className={cn("flex min-h-0 flex-col font-ui", className)}>
      <div className="group/command flex h-12 shrink-0 items-center gap-2.5 border-b border-rule/60 px-4">
        <Search
          aria-hidden="true"
          className="size-4 shrink-0 text-mute transition-colors duration-80 group-focus-within/command:text-ink"
        />
        <input
          aria-activedescendant={
            active >= 0 ? `${listId}-${active}` : undefined
          }
          aria-autocomplete="list"
          aria-controls={listId}
          aria-expanded="true"
          aria-label={label}
          autoComplete="off"
          // biome-ignore lint/a11y/noAutofocus: the menu exists to be typed into; it opens on request
          autoFocus={autoFocus}
          className="min-w-0 flex-1 bg-transparent text-ui text-ink outline-none placeholder:text-mute focus-visible:outline-none"
          onChange={(event) => onQueryChange(event.target.value)}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
          role="combobox"
          spellCheck={false}
          type="text"
          value={query}
        />
      </div>
      <div
        className={cn("min-h-0 flex-1 overflow-y-auto p-1.5", listClassName)}
        id={listId}
        role="listbox"
        aria-label={label}
      >
        <div className="relative flex flex-col" ref={containerRef}>
          {rect ? (
            <FluidHoverHighlight
              className="rounded-control"
              rect={rect}
              session={0}
            />
          ) : null}
          {items.length === 0 ? (
            <p
              aria-live="polite"
              className="px-3 py-6 text-center text-meta text-mute"
            >
              {empty}
            </p>
          ) : null}
          {items.map((item, index) => {
            const heading = item.group !== group ? item.group : undefined;
            group = item.group;
            const Icon = item.icon;
            return (
              <div className="contents" key={item.value}>
                {heading ? (
                  <p
                    aria-hidden="true"
                    className="px-3 pt-3 pb-1.5 text-label text-mute first:pt-1.5"
                  >
                    {heading}
                  </p>
                ) : null}
                {/* biome-ignore lint/a11y/useKeyWithClickEvents: the field above owns the keys, as a combobox does */}
                <div
                  aria-disabled={item.disabled || undefined}
                  aria-selected={index === active}
                  className={cn(
                    "relative z-10 flex min-h-control shrink-0 cursor-pointer items-center gap-2.5 rounded-control px-3 py-1.5 text-ui transition-colors duration-80 select-none",
                    index === active ? "text-ink" : "text-mute",
                    item.disabled && "cursor-default opacity-50",
                  )}
                  data-fluid-hover-index={index}
                  id={`${listId}-${index}`}
                  onClick={() => {
                    if (!item.disabled) onSelect(item);
                  }}
                  onMouseDown={(event) => event.preventDefault()}
                  onMouseMove={() => {
                    if (!item.disabled && index !== active) setActive(index);
                  }}
                  ref={itemRefs[index]}
                  role="option"
                  tabIndex={-1}
                >
                  {Icon ? (
                    <Icon aria-hidden className="size-4 shrink-0" />
                  ) : null}
                  <span className="grid min-w-0 flex-1">
                    <span className="truncate">{item.label}</span>
                    {item.description ? (
                      <span className="text-meta text-mute">
                        {item.description}
                      </span>
                    ) : null}
                  </span>
                  {item.trailing ? (
                    <span className="shrink-0 text-label text-mute">
                      {item.trailing}
                    </span>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <div className="flex h-10 shrink-0 items-center gap-4 border-t border-rule/60 px-4 text-label text-mute pointer-coarse:hidden">
        <span className="flex items-center gap-1.5">
          Move
          <Kbd>
            <ArrowUp />
          </Kbd>
          <Kbd>
            <ArrowDown />
          </Kbd>
        </span>
        <span className="ml-auto flex items-center gap-1.5">
          Choose
          <Kbd>
            <CornerDownLeft />
          </Kbd>
        </span>
      </div>
    </div>
  );
}

export type { CommandItem };
