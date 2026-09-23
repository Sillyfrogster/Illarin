"use client";

import { CornerDownLeft, Search } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/cn";
import type { StaffSection } from "./sections";

/** Jumps between the console's sections. Accounts, works, posts and cases join it when those sections land. */
export function ConsolePalette({
  open,
  onOpenChange,
  sections,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sections: StaffSection[];
}) {
  const router = useRouter();
  const [written, setWritten] = useState("");
  const [lit, setLit] = useState(0);

  useEffect(() => {
    if (open) {
      setWritten("");
      setLit(0);
    }
  }, [open]);

  const found = sections.filter((section) =>
    section.label.toLowerCase().includes(written.trim().toLowerCase()),
  );
  const chosen = found[Math.min(lit, found.length - 1)];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="top-[12vh] bottom-auto max-w-[34rem] p-0">
        <DialogTitle className="sr-only">Go to a section</DialogTitle>
        <DialogDescription className="sr-only">
          Type to narrow the console's sections, then press Enter.
        </DialogDescription>
        <div className="flex items-center gap-3 border-b border-rule px-4">
          <Search aria-hidden="true" className="size-4 shrink-0 text-mute" />
          <input
            autoFocus
            aria-label="Go to a section"
            className="min-h-13 w-full flex-1 border-0 bg-transparent font-ui text-ui text-ink outline-none! placeholder:text-mute"
            onChange={(event) => {
              setWritten(event.target.value);
              setLit(0);
            }}
            onKeyDown={(event) => {
              if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                event.preventDefault();
                setLit((index) => {
                  const step = event.key === "ArrowDown" ? 1 : -1;
                  return (
                    (index + step + found.length) % Math.max(found.length, 1)
                  );
                });
              }
              if (event.key === "Enter" && chosen) {
                onOpenChange(false);
                router.push(chosen.href);
              }
            }}
            placeholder="Go to a section…"
            value={written}
          />
        </div>
        <ul className="max-h-[50vh] list-none overflow-y-auto p-2">
          {found.length === 0 ? (
            <li className="px-3 py-6 text-center font-ui text-meta text-mute">
              No section by that name.
            </li>
          ) : null}
          {found.map((section, index) => (
            <li key={section.href}>
              <Link
                className={cn(
                  "flex min-h-11 items-center gap-3 rounded-control px-3 font-ui text-ui text-ink outline-offset-2",
                  section === chosen ? "bg-deep" : "",
                )}
                href={section.href}
                onClick={() => onOpenChange(false)}
                onMouseEnter={() => setLit(index)}
              >
                <section.icon aria-hidden="true" className="size-4 text-mute" />
                {section.label}
                <span className="ml-auto font-ui text-label text-mute">
                  {section.group}
                </span>
              </Link>
            </li>
          ))}
        </ul>
        <p className="flex items-center gap-2 border-t border-rule px-4 py-2.5 font-ui text-label text-mute">
          <CornerDownLeft aria-hidden="true" className="size-3.5" />
          Enter to go, Escape to close. Searching accounts and works arrives
          with those sections.
        </p>
      </DialogContent>
    </Dialog>
  );
}
