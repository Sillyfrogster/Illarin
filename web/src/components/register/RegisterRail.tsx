"use client";

import { SubtleTabs } from "@/components/ui/subtle-tabs";

export type RegisterCell<Id extends string> = {
  attention?: boolean;
  count: number | null;
  id: Id;
  name: string;
};

/** RegisterRail moves between the sections of a register, each with its count, on a row that scrolls sideways when narrow. */
export function RegisterRail<Id extends string>({
  cells,
  chosen,
  label,
  onChoose,
}: {
  cells: RegisterCell<Id>[];
  chosen: Id;
  label: string;
  onChoose: (id: Id) => void;
}) {
  return (
    <nav aria-label={label} className="-mx-[var(--gutter)] min-w-0">
      <div className="overflow-x-auto px-[calc(var(--gutter)-0.25rem)] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <SubtleTabs
          chosen={chosen}
          onChoose={onChoose}
          tabs={cells.map((cell) => ({
            value: cell.id,
            label: cell.name,
            count: cell.count ?? undefined,
            attention: cell.attention,
          }))}
        />
      </div>
    </nav>
  );
}
