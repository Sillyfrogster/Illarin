"use client";

import { Search, X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";

export function BrowseSearch({
  hint,
  id,
  label,
  onSearch,
  placeholder,
  value,
}: {
  hint?: string;
  id: string;
  label: string;
  onSearch: (query: string | undefined) => void;
  placeholder: string;
  value: string;
}) {
  const [written, setWritten] = useState(value);
  const field = useRef<HTMLInputElement>(null);

  useEffect(() => setWritten(value), [value]);

  useEffect(() => {
    function focusOnSlash(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (
        event.key !== "/" ||
        target?.closest("input, textarea, select, [contenteditable]")
      )
        return;
      event.preventDefault();
      field.current?.focus();
    }
    window.addEventListener("keydown", focusOnSlash);
    return () => window.removeEventListener("keydown", focusOnSlash);
  }, []);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSearch(written.trim() || undefined);
  }

  return (
    <search className="group/search">
      <form onSubmit={submit}>
        <label className="sr-only" htmlFor={id}>
          {label}
        </label>
        <div className="flex h-11 items-center gap-2.5 rounded-control bg-deep pr-1.5 pl-3.5 transition-shadow duration-200 focus-within:inset-ring-2 focus-within:inset-ring-accent motion-reduce:transition-none">
          <Search
            aria-hidden="true"
            className="size-4 shrink-0 text-mute"
            strokeWidth={1.8}
          />
          <input
            aria-describedby={hint ? `${id}-hint` : undefined}
            autoComplete="off"
            className="h-full min-w-0 flex-1 appearance-none border-0 bg-transparent font-ui text-ui text-ink outline-none! placeholder:text-mute [&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden"
            enterKeyHint="search"
            id={id}
            onChange={(event) => setWritten(event.target.value)}
            placeholder={placeholder}
            ref={field}
            type="search"
            value={written}
          />
          {written ? (
            <button
              aria-label="Clear the search"
              className="grid size-8 shrink-0 place-items-center rounded-[7px] text-mute outline-offset-2 hover:text-ink"
              onClick={() => {
                setWritten("");
                if (value) onSearch(undefined);
                field.current?.focus();
              }}
              type="button"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          ) : (
            <kbd className="mr-1 grid h-6 min-w-6 place-items-center rounded-[6px] bg-plane font-ui text-meta text-mute shadow-[0_1px_0_rgb(0_0_0/0.12)] group-focus-within/search:opacity-0 max-md:hidden">
              /
            </kbd>
          )}
        </div>
      </form>
      {hint ? (
        <p className="mt-1.5 font-ui text-meta text-mute" id={`${id}-hint`}>
          {hint}
        </p>
      ) : null}
    </search>
  );
}
