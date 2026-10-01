"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Tag } from "lucide-react";
import { type KeyboardEvent, useEffect, useId, useState } from "react";
import { fetchTagSuggestions, workKeys } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn, popupRow, popupSurface } from "@/lib/cn";
import { readSessionPreference } from "@/lib/nsfw-preference";
import { chooseTag, tagFragment } from "@/lib/tag-search";

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
  activeIndex: number | null;
  setActiveIndex: (index: number | null) => void;
  listId: string;
  open: boolean;
  optionId: (index: number) => string;
  rows: { value: string; count: number }[];
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
  const [lit, setLit] = useState<number | null>(null);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const typed = useAfterPause(tagFragment(written));

  // biome-ignore lint/correctness/useExhaustiveDependencies: typing again reopens the list
  useEffect(() => {
    setDismissed(false);
    setLit(null);
  }, [written]);

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
  const activeIndex = open && lit !== null && lit < rows.length ? lit : null;
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
      setLit((from + step + rows.length) % rows.length);
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
    activeIndex,
    choose,
    hint: hint && rows.length === 0,
    listId,
    open,
    optionId,
    rows,
    setActiveIndex: setLit,
  };
  return { inputProps, list };
}

/** TagSuggestionList is the popup under a search field: the tags that match, or a one-line hint while the field is empty. */
export function TagSuggestionList({
  className,
  state,
}: {
  className?: string;
  state: TagSuggestionState;
}) {
  const {
    activeIndex,
    choose,
    hint,
    listId,
    open,
    optionId,
    rows,
    setActiveIndex,
  } = state;
  if (!open) return null;

  return (
    <div
      className={cn(
        "absolute inset-x-0 top-full mt-1.5 overflow-hidden",
        popupSurface,
        className,
      )}
    >
      {hint ? (
        <p className="px-3 py-2.5 font-ui text-meta text-mute">{SEARCH_HINT}</p>
      ) : (
        <div
          aria-label="Tags"
          className="flex flex-col p-1"
          id={listId}
          onMouseLeave={() => setActiveIndex(null)}
          role="listbox"
        >
          {rows.map((row, index) => (
            // biome-ignore lint/a11y/useKeyWithClickEvents: the search field carries the keyboard through aria-activedescendant.
            // biome-ignore lint/a11y/useFocusableInteractive: focus stays in the search field, which points at the active option.
            <div
              aria-selected={activeIndex === index}
              className={popupRow}
              data-highlighted={activeIndex === index ? "" : undefined}
              id={optionId(index)}
              key={row.value}
              onClick={() => choose(row.value)}
              onMouseDown={(event) => event.preventDefault()}
              onMouseMove={() => setActiveIndex(index)}
              role="option"
            >
              <Tag aria-hidden="true" className="text-mute" />
              <span className="min-w-0 flex-1 [overflow-wrap:anywhere]">
                {row.value}
              </span>
              <span className="text-meta text-mute tabular-nums">
                {row.count}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
