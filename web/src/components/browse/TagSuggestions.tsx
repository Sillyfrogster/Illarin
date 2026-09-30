"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { Tag } from "lucide-react";
import {
  type KeyboardEvent,
  type RefObject,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { fetchTagSuggestions, workKeys } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { Elevated } from "@/lib/elevated";
import { readSessionPreference } from "@/lib/nsfw-preference";
import { popupMotionClass } from "@/lib/popup";
import { spring } from "@/lib/springs";
import { chooseTag, tagFragment } from "@/lib/tag-search";
import {
  type UseFluidHoverReturn,
  useFluidHover,
  useRegisterFluidHoverItem,
} from "@/lib/use-fluid-hover";

const TYPING_PAUSE_MS = 150;

const SEARCH_HINT = "Try tag:fantasy, or author: and a creator's handle";

function useAfterPause<T>(value: T) {
  const [paused, setPaused] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setPaused(value), TYPING_PAUSE_MS);
    return () => clearTimeout(id);
  }, [value]);
  return paused;
}

export type TagSuggestionState = {
  choose: (value: string) => void;
  hint: boolean;
  hover: UseFluidHoverReturn;
  listId: string;
  open: boolean;
  optionId: (index: number) => string;
  rows: { value: string; count: number }[];
  containerRef: RefObject<HTMLDivElement | null>;
};

/** useTagSuggestions suggests tags for the word being typed in a search field and swaps the chosen one in. */
export function useTagSuggestions({
  onChoose,
  written,
}: {
  onChoose: (query: string) => void;
  written: string;
}) {
  const { account } = useAuth();
  const listId = useId();
  const containerRef = useRef<HTMLDivElement>(null);
  const hover = useFluidHover(containerRef);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const typed = useAfterPause(tagFragment(written));

  // biome-ignore lint/correctness/useExhaustiveDependencies: typing again reopens the list
  useEffect(() => setDismissed(false), [written]);

  const suggestions = useQuery({
    enabled: typed !== null,
    placeholderData: keepPreviousData,
    queryFn: ({ signal }) =>
      fetchTagSuggestions(
        typed ?? "",
        account === null ? readSessionPreference() : undefined,
        signal,
      ),
    queryKey: [...workKeys.all, "tags", typed],
  });

  const rows = tagFragment(written) === null ? [] : (suggestions.data ?? []);
  const hint = written.trim() === "";
  const open = focused && !dismissed && (rows.length > 0 || hint);
  const activeIndex = open && rows.length > 0 ? hover.activeIndex : null;
  const optionId = (index: number) => `${listId}-${index}`;

  function choose(value: string) {
    setDismissed(true);
    onChoose(chooseTag(written, value));
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (!open || rows.length === 0) return;
    if (event.key === "Escape") {
      event.preventDefault();
      setDismissed(true);
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const step = event.key === "ArrowDown" ? 1 : -1;
      const from = activeIndex ?? (step === 1 ? -1 : 0);
      hover.setActiveIndex((from + step + rows.length) % rows.length);
    } else if (event.key === "Enter" && activeIndex !== null) {
      event.preventDefault();
      choose(rows[activeIndex].value);
    }
  }

  const inputProps = {
    "aria-activedescendant":
      activeIndex === null ? undefined : optionId(activeIndex),
    "aria-autocomplete": "list" as const,
    "aria-controls": rows.length > 0 ? listId : undefined,
    "aria-expanded": open && rows.length > 0,
    onBlur: () => setFocused(false),
    onFocus: () => setFocused(true),
    onKeyDown,
    role: "combobox" as const,
  };

  const list: TagSuggestionState = {
    choose,
    containerRef,
    hint: hint && rows.length === 0,
    hover,
    listId,
    open,
    optionId,
    rows,
  };
  return { inputProps, list };
}

function Row({
  children,
  hover,
  id,
  index,
  onChoose,
}: {
  children: React.ReactNode;
  hover: UseFluidHoverReturn;
  id: string;
  index: number;
  onChoose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useRegisterFluidHoverItem(hover.registerItem, index, ref);
  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: the search field carries the keyboard through aria-activedescendant.
    // biome-ignore lint/a11y/useFocusableInteractive: focus stays in the search field, which points at the active option.
    <div
      aria-selected={hover.activeIndex === index}
      className="relative z-10 flex min-h-control cursor-pointer items-center gap-2 rounded-control px-2 font-ui text-ui text-ink select-none"
      id={id}
      onClick={onChoose}
      onMouseDown={(event) => event.preventDefault()}
      ref={ref}
      role="option"
    >
      {children}
    </div>
  );
}

/** TagSuggestionList is the popup under a search field: the tags that match, or a one-line hint while the field is empty. */
export function TagSuggestionList({
  className,
  state,
}: {
  className?: string;
  state: TagSuggestionState;
}) {
  const { choose, containerRef, hint, hover, listId, open, optionId, rows } =
    state;
  if (!open) return null;

  return (
    <motion.div
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        "absolute inset-x-0 top-full z-90 mt-1.5",
        popupMotionClass,
        className,
      )}
      initial={{ opacity: 0, y: "var(--popup-enter-y)" }}
      transition={spring.fast}
    >
      <Elevated
        className="overflow-hidden rounded-plate"
        offset={2}
        shadowLevel={3}
      >
        {hint ? (
          <p className="px-3 py-2.5 font-ui text-meta text-mute">
            {SEARCH_HINT}
          </p>
        ) : (
          <div
            aria-label="Tags"
            className="relative flex flex-col p-1"
            id={listId}
            onMouseEnter={hover.handlers.onMouseEnter}
            onMouseLeave={hover.handlers.onMouseLeave}
            onMouseMove={hover.handlers.onMouseMove}
            ref={containerRef}
            role="listbox"
          >
            <FluidHoverHighlight className="rounded-control" hover={hover} />
            {rows.map((row, index) => (
              <Row
                hover={hover}
                id={optionId(index)}
                index={index}
                key={row.value}
                onChoose={() => choose(row.value)}
              >
                <Tag aria-hidden="true" className="size-4 shrink-0 text-mute" />
                <span className="min-w-0 flex-1 [overflow-wrap:anywhere]">
                  {row.value}
                </span>
                <span className="text-meta text-mute tabular-nums">
                  {row.count}
                </span>
              </Row>
            ))}
          </div>
        )}
      </Elevated>
    </motion.div>
  );
}
