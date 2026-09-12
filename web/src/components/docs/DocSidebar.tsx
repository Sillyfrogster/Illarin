"use client";

import { ChevronDown } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { cn } from "@/lib/cn";
import { DOCS, docHref } from "@/lib/docs/collections";

export function DocSidebar({ current }: { current: string }) {
  const here = usePathname();
  const [open, setOpen] = useState(false);

  return (
    <nav aria-label="Documentation" className="group" data-open={open}>
      <button
        aria-controls="doc-sidebar"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-3 rounded-control bg-deep px-4 font-ui text-ui font-medium text-ink outline-offset-3 lg:hidden"
        onClick={() => setOpen(!open)}
        type="button"
      >
        {current}
        <ChevronDown
          aria-hidden="true"
          className="size-4 transition-transform duration-200 group-data-[open=true]:rotate-180 motion-reduce:transition-none"
        />
      </button>
      <div
        className="mt-2 hidden flex-col gap-6 group-data-[open=true]:flex lg:mt-0 lg:flex"
        id="doc-sidebar"
      >
        {DOCS.map((collection) => (
          <div key={collection.href}>
            <p className="px-3 font-ui text-meta font-semibold text-ink">
              {collection.name}
            </p>
            <ul className="mt-1 grid list-none">
              {collection.pages.map((page) => {
                const href = docHref(collection, page);
                const chosen = href === here;
                return (
                  <li key={href}>
                    <Link
                      aria-current={chosen ? "page" : undefined}
                      className={cn(
                        "flex min-h-10 items-center rounded-control px-3 font-ui text-ui outline-offset-2 transition-colors duration-150 motion-reduce:transition-none",
                        chosen
                          ? "bg-deep font-medium text-ink"
                          : "text-mute hover:bg-deep/60 hover:text-ink",
                      )}
                      href={href}
                      onClick={() => setOpen(false)}
                    >
                      {page.title}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>
    </nav>
  );
}
