"use client";

import { useMemo, useState } from "react";
import { CommandMenu } from "@/components/ui/command-menu";
import type { ElementType } from "@/lib/api/query";
import { offerGroups } from "./composition";
import { Note } from "./fields";
import { useWorkspace } from "./state";

/** AddBlock finds a block to add by typing or browsing, one option per block and starting content. */
export function AddBlock() {
  const workspace = useWorkspace();
  const { addableBlocks, arrangement, blocks } = workspace;
  const [search, setSearch] = useState("");
  const items = useMemo(
    () =>
      offerGroups(addableBlocks, blocks, search).flatMap((group) =>
        group.offers.flatMap(({ addable, alreadyOn }) =>
          addable.choices.map((choice) => ({
            value: `${addable.definition}\u0000${choice.type}`,
            label: addable.title,
            description: addable.summary,
            group: group.title,
            trailing: alreadyOn
              ? "Already on this page"
              : addable.choices.length > 1
                ? `Starts with ${choice.label.toLowerCase()}`
                : undefined,
            disabled: alreadyOn || arrangement.busy,
          })),
        ),
      ),
    [addableBlocks, blocks, search, arrangement.busy],
  );

  if (addableBlocks.length === 0) {
    return <Note>All available blocks are already on this page.</Note>;
  }

  return (
    <CommandMenu
      className="rounded-plate border border-rule/60"
      empty="No block is named that. Try “images”, “notes” or “links”."
      items={items}
      label="Find a block"
      onQueryChange={setSearch}
      onSelect={(item) => {
        const [definition, type] = item.value.split("\u0000");
        arrangement.add(definition, type as ElementType);
        workspace.closePane();
      }}
      placeholder="Find a block"
      query={search}
    />
  );
}
