"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import type { WorkBlock } from "@/lib/api/query";
import { contentItemCount, LAYOUTS } from "@/lib/page-arrangement";
import { Note } from "./fields";
import { useWorkspace } from "./state";

export function contentDestinations(source: WorkBlock, blocks: WorkBlock[]) {
  const movable = source.elements.filter((element) => !element.pinned).length;
  if (movable === 0) return [];
  return blocks.filter(
    (block) =>
      block.id !== source.id &&
      LAYOUTS[block.layout].slots.length - block.elements.length >= movable,
  );
}

export function RemoveBlock({ block }: { block: WorkBlock }) {
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
            <p className="text-ui text-ink">Removing it deletes:</p>
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
            <Note>Removing this block deletes its content from this page.</Note>
          </>
        ) : (
          <Note>This block is empty, so nothing is lost.</Note>
        )}
      </section>

      {block.hideable || canMove ? (
        <section className="flex flex-col gap-8">
          <h3 className="font-display text-ui font-medium text-ink">
            Keep the content
          </h3>
          {block.hideable ? (
            <div className="flex flex-col gap-3">
              <p className="text-ui text-ink">Hide it from readers</p>
              <p className="text-meta text-mute">
                Everything stays in downloads and leaves the public page.
              </p>
              <Button
                className="self-start"
                disabled={arrangement.busy}
                onClick={() => {
                  arrangement.setHidden(block.id, true);
                  close();
                }}
              >
                Hide the block
              </Button>
            </div>
          ) : null}
          {canMove && destinations.length > 0 ? (
            <div className="flex flex-col gap-3">
              <p className="text-ui text-ink">
                Move the content somewhere else
              </p>
              <Field label="Move to">
                <Select
                  disabled={arrangement.busy}
                  onValueChange={setDestination}
                  options={destinations.map((candidate) => ({
                    value: candidate.id,
                    label: candidate.title,
                  }))}
                  value={destination}
                />
              </Field>
              <Button
                className="self-start"
                disabled={arrangement.busy || !destination}
                onClick={() => {
                  arrangement.moveContent(block.id, destination);
                  close();
                }}
              >
                Move it and remove this block
              </Button>
            </div>
          ) : canMove ? (
            <Note>No other block can hold this content.</Note>
          ) : null}
        </section>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        <Button
          disabled={arrangement.busy}
          onClick={() => {
            arrangement.remove(block.id);
            close();
          }}
          variant="stop"
        >
          Delete block and content
        </Button>
        <Button onClick={close} variant="ghost">
          Cancel
        </Button>
      </div>
    </div>
  );
}
