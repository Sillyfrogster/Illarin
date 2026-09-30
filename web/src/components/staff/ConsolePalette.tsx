"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { CommandMenu } from "@/components/ui/command-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
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
  const [query, setQuery] = useState("");
  const found = sections.filter((section) =>
    section.label.toLowerCase().includes(query.trim().toLowerCase()),
  );

  return (
    <Dialog
      onOpenChange={(next) => {
        if (next) setQuery("");
        onOpenChange(next);
      }}
      open={open}
    >
      <DialogContent
        className="max-w-[34rem]"
        position="top"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">Go to a section</DialogTitle>
        <DialogDescription className="sr-only">
          Type to narrow the console's sections, then press Enter.
        </DialogDescription>
        <CommandMenu
          autoFocus
          empty="No section by that name."
          items={found.map((section) => ({
            value: section.href,
            label: section.label,
            icon: section.icon,
            trailing: section.group,
          }))}
          label="Go to a section"
          listClassName="max-h-[50vh]"
          onQueryChange={setQuery}
          onSelect={(item) => {
            onOpenChange(false);
            router.push(item.value);
          }}
          placeholder="Go to a section…"
          query={query}
        />
      </DialogContent>
    </Dialog>
  );
}
