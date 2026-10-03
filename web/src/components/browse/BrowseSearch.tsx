"use client";

import { Search, X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Kbd } from "@/components/ui/kbd";
import { TagSuggestionList, useTagSuggestions } from "./TagSuggestions";

/** BrowseSearch is a list's search field, focused by "/"; on the catalog it also suggests tags as you type. */
export function BrowseSearch({
  hint,
  id,
  label,
  onSearch,
  placeholder,
  tags = false,
  value,
}: {
  hint?: string;
  id: string;
  label: string;
  onSearch: (query: string | undefined) => void;
  placeholder: string;
  tags?: boolean;
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

  const suggestions = useTagSuggestions({
    onChoose: (query) => {
      setWritten(query);
      onSearch(query.trim() || undefined);
    },
    written: tags ? written : "",
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSearch(written.trim() || undefined);
  }

  return (
    <search className="group/search relative">
      <form onSubmit={submit}>
        <label className="sr-only" htmlFor={id}>
          {label}
        </label>
        <InputGroup>
          <InputGroupAddon>
            <Search aria-hidden="true" />
          </InputGroupAddon>
          <InputGroupInput
            {...(tags ? suggestions.inputProps : {})}
            aria-describedby={hint ? `${id}-hint` : undefined}
            autoComplete="off"
            className="[&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden"
            enterKeyHint="search"
            id={id}
            onChange={(event) => setWritten(event.target.value)}
            placeholder={placeholder}
            ref={field}
            type="search"
            value={written}
          />
          <InputGroupAddon align="inline-end">
            {written ? (
              <Button
                onClick={() => {
                  setWritten("");
                  if (value) onSearch(undefined);
                  field.current?.focus();
                }}
                size="icon-compact"
                variant="ghost"
              >
                <X aria-hidden="true" />
                <span className="sr-only">Clear the search</span>
              </Button>
            ) : (
              <Kbd className="group-focus-within/search:opacity-0 max-md:hidden">
                /
              </Kbd>
            )}
          </InputGroupAddon>
        </InputGroup>
      </form>
      {tags ? <TagSuggestionList state={suggestions.list} /> : null}
      {hint ? (
        <p className="mt-1.5 font-ui text-meta text-mute" id={`${id}-hint`}>
          {hint}
        </p>
      ) : null}
    </search>
  );
}
