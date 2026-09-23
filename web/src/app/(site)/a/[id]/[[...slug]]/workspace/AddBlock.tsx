"use client";

import { Check, Plus, Search } from "lucide-react";
import { useId, useMemo, useState } from "react";
import type { ElementType } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { offerGroups } from "./composition";
import { AddAction, Note } from "./fields";
import { useWorkspace } from "./state";

const SEARCH_FROM = 6;

const CHOICE =
  "inline-flex min-h-11 items-center rounded-control bg-deep px-3.5 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45";

export function AddBlock() {
  const workspace = useWorkspace();
  const { addableBlocks, arrangement, blocks } = workspace;
  const [search, setSearch] = useState("");
  const searchId = useId();
  const groups = useMemo(
    () => offerGroups(addableBlocks, blocks, search),
    [addableBlocks, blocks, search],
  );

  function add(definition: string, elementType: ElementType) {
    arrangement.add(definition, elementType);
    workspace.closePane();
  }

  if (addableBlocks.length === 0) {
    return <Note>All available blocks are already on this page.</Note>;
  }

  return (
    <div className="flex flex-col gap-8">
      {addableBlocks.length >= SEARCH_FROM ? (
        <div className="flex items-center gap-2 rounded-control bg-deep px-3">
          <Search aria-hidden="true" className="shrink-0 text-mute" size={16} />
          <label className="sr-only" htmlFor={searchId}>
            Search the blocks
          </label>
          <input
            className="min-h-11 w-full border-0 bg-transparent text-ui text-ink outline-offset-3 placeholder:text-mute"
            id={searchId}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search blocks"
            type="search"
            value={search}
          />
        </div>
      ) : null}

      {groups.length === 0 ? (
        <Note>No block is named that. Try “images”, “notes” or “links”.</Note>
      ) : null}

      {groups.map((group) => (
        <section key={group.key}>
          <h3 className="mb-3 font-display text-ui font-medium text-ink">
            {group.title}
          </h3>
          <ul className="flex list-none flex-col gap-1.5">
            {group.offers.map(({ addable, alreadyOn }) => (
              <li
                className={cn(
                  "rounded-plate p-3.5",
                  alreadyOn ? "bg-deep/50" : "bg-deep",
                )}
                key={addable.definition}
              >
                <p
                  className={cn(
                    "text-ui font-medium text-ink",
                    alreadyOn && "opacity-60",
                  )}
                >
                  {addable.title}
                </p>
                <p
                  className={cn(
                    "mt-1 text-meta text-mute",
                    alreadyOn && "opacity-60",
                  )}
                >
                  {addable.summary}
                </p>
                {alreadyOn ? (
                  <p className="mt-2.5 flex items-center gap-1.5 text-label text-mute">
                    <Check aria-hidden="true" size={14} />
                    Already on this page
                  </p>
                ) : addable.choices.length === 1 ? (
                  <div className="mt-3">
                    <AddAction
                      disabled={arrangement.busy}
                      onClick={() =>
                        add(addable.definition, addable.choices[0].type)
                      }
                    >
                      Add block
                    </AddAction>
                  </div>
                ) : (
                  <div className="mt-3">
                    <p className="mb-2 text-label text-mute">Initial content</p>
                    <div className="flex flex-wrap gap-1.5">
                      {addable.choices.map((choice) => (
                        <button
                          className={CHOICE}
                          disabled={arrangement.busy}
                          key={choice.type}
                          onClick={() => add(addable.definition, choice.type)}
                          type="button"
                        >
                          <Plus
                            aria-hidden="true"
                            className="mr-1.5"
                            size={14}
                          />
                          {choice.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
