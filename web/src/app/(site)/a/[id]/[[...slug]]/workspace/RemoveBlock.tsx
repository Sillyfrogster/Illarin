"use client";

import { useState } from "react";
import type { AssetBlock } from "@/lib/api/query";
import { contentItemCount, LAYOUTS } from "@/lib/page-arrangement";
import { ChoiceField, Field, Note } from "./fields";
import { useWorkspace } from "./state";

const KEEP =
  "inline-flex min-h-11 items-center self-start rounded-control bg-deep px-4 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45";

/** The blocks with room for everything one block would leave behind. */
export function contentDestinations(source: AssetBlock, blocks: AssetBlock[]) {
  const movable = source.elements.filter((element) => !element.pinned).length;
  if (movable === 0) return [];
  return blocks.filter(
    (block) =>
      block.id !== source.id &&
      LAYOUTS[block.layout].slots.length - block.elements.length >= movable,
  );
}

/** What a removal takes with it, and the ways to keep it instead. */
export function RemoveBlock({ block }: { block: AssetBlock }) {
  const workspace = useWorkspace();
  const { arrangement } = workspace;
  const destinations = contentDestinations(block, workspace.blocks);
  const [destination, setDestination] = useState(destinations[0]?.id ?? "");
  const losses = block.elements
    .map((element) => ({ count: contentItemCount(element), element }))
    .filter(({ count }) => count > 0);
  const holdsContent = losses.length > 0;
  const movable = block.elements.filter((element) => !element.pinned);
  const canMove = holdsContent && movable.length > 0;

  function close() {
    workspace.closePane();
  }

  return (
    <div className="flex flex-col gap-10">
      <section className="flex flex-col gap-3">
        {holdsContent ? (
          <>
            <p className="text-ui text-ink">
              This takes the following with it:
            </p>
            <ul className="flex list-none flex-col gap-2">
              {losses.map(({ count, element }) => (
                <li key={element.id}>
                  <strong className="text-ui font-medium text-ink">
                    {element.label}
                  </strong>
                  <span className="mt-0.5 block text-meta text-mute">
                    {count} {count === 1 ? "item" : "items"}
                    {element.role
                      ? `, and it is what a download reads for ${element.label.toLowerCase()}`
                      : " of page content"}
                  </span>
                </li>
              ))}
            </ul>
            <Note>
              There is nowhere else this content is kept. The block is where it
              lives.
            </Note>
          </>
        ) : (
          <Note>This block is empty, so nothing is lost.</Note>
        )}
      </section>

      {block.hideable || canMove ? (
        <section className="flex flex-col gap-8">
          <h3 className="font-display text-ui font-medium text-ink">
            Or keep it
          </h3>
          {block.hideable ? (
            <div className="flex flex-col gap-3">
              <p className="text-ui text-ink">Hide it from readers</p>
              <p className="text-meta text-mute">
                Everything stays in downloads and leaves the public page.
              </p>
              <button
                className={KEEP}
                disabled={arrangement.busy}
                onClick={() => {
                  arrangement.setHidden(block.id, true);
                  close();
                }}
                type="button"
              >
                Hide the block
              </button>
            </div>
          ) : null}
          {canMove && destinations.length > 0 ? (
            <div className="flex flex-col gap-3">
              <p className="text-ui text-ink">
                Move the content somewhere else
              </p>
              <Field label="The block that keeps it">
                <ChoiceField
                  disabled={arrangement.busy}
                  onChange={(event) => setDestination(event.target.value)}
                  value={destination}
                >
                  {destinations.map((candidate) => (
                    <option key={candidate.id} value={candidate.id}>
                      {candidate.title}
                    </option>
                  ))}
                </ChoiceField>
              </Field>
              <button
                className={KEEP}
                disabled={arrangement.busy || !destination}
                onClick={() => {
                  arrangement.moveContent(block.id, destination);
                  close();
                }}
                type="button"
              >
                Move it and remove this block
              </button>
            </div>
          ) : canMove ? (
            <Note>No other block has room for these elements yet.</Note>
          ) : null}
        </section>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        <button
          className="inline-flex min-h-11 items-center rounded-control bg-stop px-5 text-ui font-medium text-on-stop outline-offset-3 hover:opacity-90 disabled:opacity-45"
          disabled={arrangement.busy}
          onClick={() => {
            arrangement.remove(block.id);
            close();
          }}
          type="button"
        >
          Remove and delete
        </button>
        <button
          className="inline-flex min-h-11 items-center rounded-control px-4 text-ui font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink"
          onClick={close}
          type="button"
        >
          Keep it
        </button>
      </div>
    </div>
  );
}
