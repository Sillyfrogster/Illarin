"use client";

import { Search, X } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { cn } from "@/lib/cn";

/** The catalog's one question. It is wide on Browse and inline on a creator's page. */
export function CatalogSearch({
  hint,
  id,
  label,
  onSearch,
  placeholder,
  size = "inline",
  value,
}: {
  hint?: string;
  id: string;
  label: string;
  onSearch: (query: string | undefined) => void;
  placeholder: string;
  size?: "hero" | "inline";
  value: string;
}) {
  const [written, setWritten] = useState(value);
  const hero = size === "hero";

  useEffect(() => setWritten(value), [value]);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSearch(written.trim() || undefined);
  }

  return (
    <search>
      <form onSubmit={submit}>
        <label className="sr-only" htmlFor={id}>
          {label}
        </label>
        <div
          className={cn(
            "flex items-center gap-3 rounded-plate bg-plane inset-ring inset-ring-rule transition-shadow duration-200 focus-within:inset-ring-2 focus-within:inset-ring-accent motion-reduce:transition-none",
            hero ? "h-14 pr-2 pl-5" : "h-12 pr-1.5 pl-4",
          )}
        >
          <Search
            aria-hidden="true"
            className={cn("shrink-0 text-mute", hero ? "size-5" : "size-4")}
            strokeWidth={1.6}
          />
          <input
            className={cn(
              "h-full min-w-0 flex-1 appearance-none border-0 bg-transparent text-ink outline-none! placeholder:text-mute",
              "[&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden",
              hero ? "text-lede" : "text-ui",
            )}
            autoComplete="off"
            id={id}
            onChange={(event) => setWritten(event.target.value)}
            placeholder={placeholder}
            type="search"
            value={written}
          />
          {written ? (
            <button
              aria-label="Clear the search"
              className="grid size-11 shrink-0 place-items-center rounded-control text-mute outline-offset-2 hover:text-ink"
              onClick={() => {
                setWritten("");
                if (value) onSearch(undefined);
              }}
              type="button"
            >
              <X aria-hidden="true" className="size-4" />
            </button>
          ) : null}
          <button
            className={cn(
              "shrink-0 rounded-control bg-action font-ui font-medium text-on-accent outline-offset-2 transition-opacity duration-200 hover:opacity-90 motion-reduce:transition-none",
              hero ? "min-h-11 px-6 text-ui" : "min-h-9 px-4 text-meta",
            )}
            type="submit"
          >
            Search
          </button>
        </div>
      </form>
      {hint ? <p className="mt-3 font-ui text-meta text-mute">{hint}</p> : null}
    </search>
  );
}
