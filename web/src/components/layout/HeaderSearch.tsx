"use client";

import { Search } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, useEffect, useRef, useState } from "react";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Kbd } from "@/components/ui/kbd";
import { buildBrowseHref, readBrowseFilters } from "@/lib/browse-url";

const SEARCH_ID = "header-search";

/** HeaderSearch runs Browse's search from any page; on Browse it keeps the type and features already chosen. */
export function HeaderSearch() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const onBrowse = pathname === "/browse";
  const current = onBrowse ? (params.get("q") ?? "") : "";
  const [written, setWritten] = useState(current);
  const field = useRef<HTMLInputElement>(null);

  useEffect(() => setWritten(current), [current]);

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
    const q = written.trim() || undefined;
    const kept = onBrowse
      ? readBrowseFilters(Object.fromEntries(params.entries()))
      : {};
    router.push(buildBrowseHref({ ...kept, q }), { scroll: false });
  }

  return (
    <search className="group/search min-w-0 max-md:order-last max-md:w-full md:w-64 lg:w-72">
      <form onSubmit={submit}>
        <label className="sr-only" htmlFor={SEARCH_ID}>
          Search works
        </label>
        <InputGroup>
          <InputGroupAddon>
            <Search aria-hidden="true" />
          </InputGroupAddon>
          <InputGroupInput
            autoComplete="off"
            className="[&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden"
            enterKeyHint="search"
            id={SEARCH_ID}
            onChange={(event) => setWritten(event.target.value)}
            placeholder="Search works, or tag:fantasy"
            ref={field}
            type="search"
            value={written}
          />
          <InputGroupAddon align="inline-end">
            <Kbd className="group-focus-within/search:opacity-0 max-md:hidden">
              /
            </Kbd>
          </InputGroupAddon>
        </InputGroup>
      </form>
    </search>
  );
}
