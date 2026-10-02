"use client";

import { Plus } from "lucide-react";
import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Arrive } from "@/components/ui/arrive";
import { Badge } from "@/components/ui/badge";
import type {
  BrowseType,
  WorkBlock,
  WorkElement,
  WorkImage,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  BLOCK_GRID_GAP_PX,
  elementTracks,
  placeBlocks,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import { blockCounts } from "@/lib/work-block-heading";
import {
  coreBlockTitles,
  rendersOnThePage,
  splitWorkPageContent,
  workHoldsNothing,
} from "@/lib/work-page-content";
import { ContentsBar } from "./ContentsBar";
import { ElementBody } from "./ElementBody";
import { EmptyPage, EmptyPageInvitation } from "./QuietPage";
import { useSuggestedWidths } from "./use-suggested-widths";
import { useAltitudeShift } from "./workspace/Altitudes";
import { BlockAudience } from "./workspace/BlockAudience";
import {
  ColumnGuides,
  Frame,
  LiftCard,
  MapTile,
  Seam,
} from "./workspace/BlockChrome";
import { EditableElementSection } from "./workspace/EditableElement";
import { EditableText } from "./workspace/EditableText";
import { GhostBlock, TextTarget } from "./workspace/ShelfTargets";
import { useShelf } from "./workspace/shelf";
import { withGhost } from "./workspace/shelf-places";
import { useWorkspace } from "./workspace/state";
import { useArranging } from "./workspace/use-arranging";

function returnToBlock(blockId: string) {
  const anchor = `block-${blockId}`;
  document.getElementById(anchor)?.scrollIntoView({ block: "start" });
  window.location.hash = anchor;
}

export function WorkBlocks({
  images,
  isOwner,
  type,
  shellClassName,
}: {
  images: WorkImage[];
  isOwner: boolean;
  type: BrowseType;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const rowsNode = useRef<HTMLDivElement | null>(null);
  const known = useRef<Set<string> | null>(null);

  const blocks = workspace.blocks;
  const writing = isOwner && workspace.editing;
  const hands = writing && workspace.look === "hands";
  const mapped =
    writing &&
    workspace.look === "altitudes" &&
    workspace.altitude === "arrange";
  const arranging = useArranging();
  const shelf = useShelf();
  const moving = shelf.dragging ?? shelf.pending?.piece ?? null;
  const landing = shelf.pending?.target ?? shelf.target;
  const ghostAt =
    writing && moving && landing && "position" in landing
      ? landing.position
      : null;

  useEffect(() => {
    const seen = known.current;
    const fresh = seen ? blocks.find((block) => !seen.has(block.id)) : null;
    known.current = new Set(blocks.map((block) => block.id));
    if (fresh && !shelf.arrivesQuietly()) returnToBlock(fresh.id);
  }, [blocks, shelf]);

  const { publicBlocks, modelContent: disclosedModelContent } = useMemo(
    () => splitWorkPageContent(blocks),
    [blocks],
  );
  const modelContent = writing ? [] : disclosedModelContent;
  const [rowsRef, availableWidth] = useMeasuredWidth<HTMLDivElement>();
  const setRowsRef = useCallback(
    (node: HTMLDivElement | null) => {
      rowsNode.current = node;
      rowsRef(node);
    },
    [rowsRef],
  );
  const packable: Array<WorkBlock & { empty?: boolean }> = writing
    ? arranging.shown
    : publicBlocks.filter(rendersOnThePage);
  const placed = placeBlocks(
    withGhost<(typeof packable)[number] | typeof GHOST>(
      packable,
      ghostAt,
      GHOST,
    ),
    { availableWidth },
  );
  const invited = writing && workHoldsNothing(blocks);
  const contentsBlocks = useMemo(
    () => (writing ? blocks : publicBlocks.filter(rendersOnThePage)),
    [blocks, publicBlocks, writing],
  );
  const suggestedWidths = useSuggestedWidths({
    availableWidth,
    blocks,
    paused: !writing || workspace.cursor !== null || arranging.lift !== null,
    rows: rowsNode,
  });
  const rows = seamRows(placed);
  const shift = useAltitudeShift();

  return (
    <>
      {mapped ? null : (
        <ContentsBar
          blocks={contentsBlocks}
          shellClassName={shellClassName}
          writing={writing && !hands}
        />
      )}

      <div
        className={cn(
          shellClassName,
          mapped ? "pt-6" : hands ? "pt-24" : "pt-10",
        )}
        data-shelf-page={writing ? blocks.length : undefined}
      >
        {writing && !mapped ? (
          <p className="mb-8 rounded-control bg-deep p-3 text-meta text-mute md:hidden">
            On a phone, every block is full width.
          </p>
        ) : null}
        {invited ? (
          <EmptyPageInvitation
            canAdd={workspace.addableBlocks.length > 0}
            coreBlocks={coreBlockTitles(blocks)}
            type={type}
          />
        ) : placed.length === 0 ? (
          <EmptyPage type={type} />
        ) : null}
        {placed.length === 0 ? null : (
          <div
            className={cn(
              "relative isolate grid grid-cols-1 items-start md:grid-cols-12 md:gap-x-[var(--block-grid-gap)]",
              hands
                ? "gap-20 md:gap-y-[var(--seam-gap)]"
                : mapped
                  ? "gap-10 md:gap-y-12"
                  : "gap-14 md:gap-y-[5.5rem]",
            )}
            data-block-grid
            ref={setRowsRef}
            style={
              {
                "--block-grid-gap": `${BLOCK_GRID_GAP_PX}px`,
                "--seam-gap": "7rem",
              } as CSSProperties
            }
          >
            {writing ? (
              <ColumnGuides shown={arranging.stretch !== null || mapped} />
            ) : null}
            {placed.map(({ block, columns, startColumn, row, place }) => {
              if ("ghost" in block)
                return (
                  <Arrive
                    className="col-span-full min-w-0 md:[grid-column:var(--block-start)_/_span_var(--block-columns)] md:[grid-row:var(--block-row)]"
                    key={block.id}
                    layout={shelf.shifting}
                    quiet
                    style={
                      {
                        "--block-columns": columns,
                        "--block-row": row,
                        "--block-start": startColumn,
                      } as CSSProperties
                    }
                  >
                    {moving ? <GhostBlock piece={moving} /> : null}
                  </Arrive>
                );
              const position = blocks.findIndex((one) => one.id === block.id);
              const lifted = arranging.lift?.blockId === block.id;
              const folded = hands && arranging.lift !== null;
              const suggestion = suggestedWidths[block.id];
              return (
                <Arrive
                  className={cn(
                    "col-span-full min-w-0 md:[grid-column:var(--block-start)_/_span_var(--block-columns)] md:[grid-row:var(--block-row)]",
                    arranging.stretch?.blockId === block.id && "relative z-20",
                  )}
                  key={block.id}
                  layout={
                    shelf.shifting || (writing && arranging.lift !== null)
                  }
                  quiet={writing}
                  place={writing ? 0 : place}
                  style={
                    {
                      "--block-columns": columns,
                      "--block-row": row,
                      "--block-start": startColumn,
                    } as CSSProperties
                  }
                >
                  <div data-shelf-index={position}>
                    <article
                      className={cn(
                        "group/block relative min-w-0 scroll-mt-[calc(var(--header-height)+5rem)] [container-name:block] [container-type:inline-size]",
                        hands &&
                          "after:pointer-events-none after:absolute after:-inset-x-2 after:-inset-y-4 after:rounded-card after:ring-1 after:ring-rule/70 after:transition-[box-shadow] after:duration-200 after:content-[''] hover:after:ring-accent/50 focus-within:after:ring-accent/50",
                        arranging.stretch?.blockId === block.id && "z-20",
                        hands &&
                          arranging.stretch?.blockId === block.id &&
                          "after:!ring-2 after:!ring-accent",
                        mapped &&
                          "rounded-card bg-plane px-8 pt-24 pb-8 ring-1 ring-ink/8 transition-[box-shadow] duration-200 hover:ring-2 hover:ring-accent/60",
                        mapped &&
                          arranging.stretch?.blockId === block.id &&
                          "ring-2 ring-accent",
                        lifted && "opacity-40 after:!ring-2 after:!ring-accent",
                        lifted && mapped && "bg-accent-wash ring-2 ring-accent",
                        shelf.glowing === block.id &&
                          "after:!ring-2 after:!ring-accent",
                        writing &&
                          block.hidden &&
                          !mapped &&
                          "bg-deep/60 px-5 pt-6 pb-7",
                        mapped && block.hidden && "bg-deep",
                      )}
                      data-arrange-id={writing ? block.id : undefined}
                      data-block-id={block.id}
                      data-hidden={writing && block.hidden ? true : undefined}
                      id={`block-${block.id}`}
                      style={
                        writing
                          ? ({
                              viewTransitionName: `block-${block.id}`,
                              viewTransitionClass: "edit-block",
                            } as CSSProperties)
                          : undefined
                      }
                    >
                      {mapped ? (
                        <MapTile
                          arranging={arranging}
                          block={block}
                          onOpen={() => shift("write", block.id)}
                          position={position}
                          startColumn={startColumn}
                          suggestion={suggestion}
                          total={blocks.length}
                        />
                      ) : hands ? (
                        <Frame
                          arranging={arranging}
                          block={block}
                          position={position}
                          startColumn={startColumn}
                          suggestion={suggestion}
                          total={blocks.length}
                        />
                      ) : null}
                      <header
                        className={cn(
                          "mb-5 flex flex-wrap items-end justify-between gap-x-6 gap-y-3.5",
                          writing && block.hidden ? "opacity-50" : null,
                        )}
                      >
                        <div className="flex min-w-0 flex-1 basis-45 flex-wrap items-baseline gap-x-2.5 gap-y-1">
                          <BlockTitle block={block} live={writing && !mapped} />
                          {writing && block.required && !mapped ? (
                            <Badge className="shrink-0">
                              {block.hideable ? "Required" : "Always shown"}
                            </Badge>
                          ) : null}
                          <BlockCounts elements={block.elements} />
                        </div>
                      </header>
                      {writing && !mapped ? (
                        <BlockAudience block={block} />
                      ) : null}
                      <div
                        className={cn(
                          "grid gap-x-8 gap-y-7 [grid-template-columns:var(--element-tracks,minmax(0,1fr))] max-md:![grid-template-columns:minmax(0,1fr)]",
                          writing && block.hidden ? "opacity-50" : null,
                          (mapped || folded) &&
                            "overflow-hidden [mask-image:linear-gradient(to_bottom,black_55%,transparent)]",
                          mapped && "max-h-48 md:max-h-[56rem]",
                          folded && "max-h-40",
                          mapped && "pointer-events-none",
                        )}
                        data-block-content
                        inert={mapped || undefined}
                        style={
                          {
                            "--element-tracks": elementTracks(
                              block.layout,
                              block.elements.length,
                            ),
                          } as CSSProperties
                        }
                      >
                        {block.elements.map((element) => (
                          <div
                            data-empty={element.isEmpty ? true : undefined}
                            key={element.id}
                          >
                            {writing &&
                            !mapped &&
                            element.type === "prose" &&
                            !element.fromFile ? (
                              <TextTarget
                                blockId={block.id}
                                elementId={element.id}
                              >
                                <EditableElementSection
                                  block={block}
                                  element={element}
                                  images={images}
                                  markEmpty={!invited}
                                />
                              </TextTarget>
                            ) : writing && !mapped ? (
                              <EditableElementSection
                                block={block}
                                element={element}
                                images={images}
                                markEmpty={!invited}
                              />
                            ) : (
                              <ElementBody
                                blockElements={block.elements.length}
                                blockTitle={block.title}
                                element={element}
                                images={images}
                                isOwner={false}
                                markEmpty={!invited}
                              />
                            )}
                          </div>
                        ))}
                      </div>
                    </article>
                  </div>
                </Arrive>
              );
            })}
            {hands && arranging.lift === null
              ? rows.map(({ row, at, last }) => (
                  <div
                    className="contents"
                    key={`seam-${row}-${last ? "end" : "start"}`}
                    style={{ "--seam-row": row } as CSSProperties}
                  >
                    <Seam at={at} last={last} />
                  </div>
                ))
              : null}
            {mapped && workspace.addableBlocks.length > 0 ? (
              <button
                className="col-span-full flex min-h-40 items-center justify-center gap-3 rounded-card bg-fill text-title font-medium text-mute transition-colors duration-150 hover:bg-fill-hover hover:text-ink md:[grid-column:1/-1]"
                onClick={() => workspace.openPane({ kind: "add-block" })}
                type="button"
              >
                <Plus aria-hidden="true" className="size-8" />
                Add a block
              </button>
            ) : null}
          </div>
        )}
        {modelContent.length > 0 ? (
          <Accordion
            className="mt-section rounded-plate bg-deep/70 px-5 py-3"
            collapsible
            type="single"
          >
            <AccordionItem value="model">
              <AccordionTrigger trailing="Sent to the model with every chat.">
                Model instructions
              </AccordionTrigger>
              <AccordionContent className="grid gap-8 pt-5 pb-2">
                {modelContent.map(({ element }) => (
                  <ElementBody
                    element={element}
                    images={images}
                    isOwner={false}
                    key={element.id}
                  />
                ))}
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        ) : null}
      </div>
      {writing ? <LiftCard arranging={arranging} /> : null}
    </>
  );
}

/** seamRows names where a block can be added between rows: before the first block of each row, and after the last row. */
function seamRows<T extends { block: { id: string } }>(
  placed: Array<T & { row: number }>,
): Array<{ row: number; at: number; last: boolean }> {
  const seams: Array<{ row: number; at: number; last: boolean }> = [];
  let count = 0;
  let row = 0;
  for (const one of placed) {
    if (one.row !== row) {
      row = one.row;
      seams.push({ at: count, last: false, row });
    }
    if (!("ghost" in one.block)) count += 1;
  }
  if (row > 0) seams.push({ at: count, last: true, row });
  return seams;
}

const GHOST = { id: "ghost", ghost: true, width: "full" } as const;

function BlockTitle({ block, live }: { block: WorkBlock; live: boolean }) {
  const workspace = useWorkspace();
  const cursor = `block:${block.id}:title`;
  return (
    <EditableText
      active={workspace.cursor === cursor}
      activate={() => workspace.setCursor(cursor)}
      as="h2"
      className="font-display text-title font-medium tracking-tight text-ink [overflow-wrap:anywhere]"
      done={() => workspace.setCursor(null)}
      label={`Heading of ${block.title}`}
      live={live}
      onChange={(title) =>
        workspace.setBlocks(
          workspace.blocks.map((item) =>
            item.id === block.id
              ? { ...item, title, titleIsDefault: title.trim() === "" }
              : item,
          ),
        )
      }
      placeholder="Name this block"
      singleLine
      value={block.title}
    />
  );
}

function BlockCounts({ elements }: { elements: WorkElement[] }) {
  const counts = blockCounts(elements);
  if (!counts) return null;
  return <p className="basis-full text-label text-mute">{counts}</p>;
}
