"use client";

import { Plus } from "lucide-react";
import {
  type CSSProperties,
  memo,
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
  type BlockWidth,
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
import { ColumnGuides, MapTile } from "./workspace/BlockChrome";
import { EditableElementSection } from "./workspace/EditableElement";
import { EditableText } from "./workspace/EditableText";
import { previewElement } from "./workspace/map-preview";
import { GhostBlock, TextTarget } from "./workspace/ShelfTargets";
import { useShelf } from "./workspace/shelf";
import { withGhost } from "./workspace/shelf-places";
import {
  useWorkspace,
  useWorkspaceActions,
  useWorkspaceFocus,
} from "./workspace/state";
import {
  type ArrangeHandlers,
  useArranging,
  useReflow,
} from "./workspace/use-arranging";

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
  const mapped = writing && workspace.altitude === "arrange";
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
    // Measured on the written page and kept for the map, where blocks are cut short
    paused: !writing || mapped || workspace.cursor !== null,
    rows: rowsNode,
  });
  const shift = useAltitudeShift();
  const { pressBlock, pressEdge, pressGrip } = arranging;
  const handlers = useMemo(
    () => ({ pressBlock, pressEdge, pressGrip }),
    [pressBlock, pressEdge, pressGrip],
  );
  useReflow(
    placed.map((one) => `${one.block.id}:${one.columns}`).join(" "),
    mapped,
  );
  const stretched = arranging.stretch
    ? placed.find((one) => one.block.id === arranging.stretch?.blockId)
    : undefined;

  return (
    <>
      {mapped ? null : (
        <ContentsBar
          blocks={contentsBlocks}
          shellClassName={shellClassName}
          writing={writing}
        />
      )}

      <div
        className={cn(shellClassName, mapped ? "pt-6" : "pt-10")}
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
              mapped ? "gap-10 md:gap-y-12" : "gap-14 md:gap-y-[5.5rem]",
            )}
            data-block-grid
            ref={setRowsRef}
            style={
              {
                "--block-grid-gap": `${BLOCK_GRID_GAP_PX}px`,
              } as CSSProperties
            }
          >
            {mapped ? (
              <ColumnGuides
                span={
                  stretched
                    ? {
                        columns: stretched.columns,
                        start: stretched.startColumn,
                      }
                    : null
                }
              />
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
              return (
                <BlockView
                  block={block}
                  columns={columns}
                  glowing={shelf.glowing === block.id}
                  handlers={handlers}
                  images={images}
                  invited={invited}
                  key={block.id}
                  lifted={arranging.lifted === block.id}
                  mapped={mapped}
                  place={writing ? 0 : place}
                  position={position}
                  row={row}
                  shift={shift}
                  shifting={shelf.shifting && !mapped}
                  startColumn={startColumn}
                  stretching={arranging.stretch?.blockId === block.id}
                  suggestion={suggestedWidths[block.id]}
                  total={blocks.length}
                  writing={writing}
                />
              );
            })}
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
    </>
  );
}

const GHOST = { id: "ghost", ghost: true, width: "full" } as const;

type BlockViewProps = {
  block: WorkBlock;
  columns: number;
  glowing: boolean;
  handlers: ArrangeHandlers;
  images: WorkImage[];
  invited: boolean;
  lifted: boolean;
  mapped: boolean;
  place: number;
  position: number;
  row: number;
  shift: ReturnType<typeof useAltitudeShift>;
  shifting: boolean;
  startColumn: number;
  stretching: boolean;
  suggestion?: BlockWidth;
  total: number;
  writing: boolean;
};

/** BlockView is one block on the page, drawn again only when that block or its own state changes, so writing in one block leaves the rest alone. */
const BlockView = memo(function BlockView({
  block,
  columns,
  glowing,
  handlers,
  images,
  invited,
  lifted,
  mapped,
  place,
  position,
  row,
  shift,
  shifting,
  startColumn,
  stretching,
  suggestion,
  total,
  writing,
}: BlockViewProps) {
  const live = writing && !mapped;
  return (
    <Arrive
      className={cn(
        "col-span-full min-w-0 md:[grid-column:var(--block-start)_/_span_var(--block-columns)] md:[grid-row:var(--block-row)]",
        stretching && "relative z-20",
      )}
      layout={shifting}
      place={place}
      quiet={writing}
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
            mapped &&
              "rounded-card bg-plane px-5 pt-16 pb-5 ring-1 md:px-8 md:pt-24 md:pb-8 ring-ink/10 transition-[box-shadow,background-color] duration-200 hover:ring-2 hover:ring-accent/70",
            mapped && stretching && "z-20 ring-2 ring-accent",
            mapped && block.hidden && "bg-deep",
            lifted &&
              "bg-accent-wash outline-2 outline-offset-0 outline-accent outline-dashed ring-0 hover:ring-0 [&>*]:invisible",
            glowing && "after:!ring-2 after:!ring-accent",
            live && block.hidden && "bg-deep/60 px-5 pt-6 pb-7",
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
              arranging={handlers}
              block={block}
              onOpen={() => shift("write", block.id)}
              position={position}
              startColumn={startColumn}
              stretching={stretching}
              suggestion={suggestion}
              total={total}
            />
          ) : null}
          <header
            className={cn(
              "mb-5 flex flex-wrap items-end justify-between gap-x-6 gap-y-3.5",
              writing && block.hidden ? "opacity-50" : null,
            )}
          >
            <div className="flex min-w-0 flex-1 basis-45 flex-wrap items-baseline gap-x-2.5 gap-y-1">
              <BlockTitle block={block} live={live} />
              {live && block.required ? (
                <Badge className="shrink-0">
                  {block.hideable ? "Required" : "Always shown"}
                </Badge>
              ) : null}
              <BlockCounts elements={block.elements} />
            </div>
          </header>
          {live ? <BlockAudience block={block} /> : null}
          <div
            className={cn(
              "grid gap-x-8 gap-y-7 [grid-template-columns:var(--element-tracks,minmax(0,1fr))] max-md:![grid-template-columns:minmax(0,1fr)]",
              writing && block.hidden ? "opacity-50" : null,
              mapped &&
                "pointer-events-none max-h-48 overflow-hidden [mask-image:linear-gradient(to_bottom,black_55%,transparent)] md:max-h-[34rem]",
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
                {live && element.type === "prose" && !element.fromFile ? (
                  <TextTarget blockId={block.id} elementId={element.id}>
                    <EditableElementSection
                      block={block}
                      element={element}
                      images={images}
                      markEmpty={!invited}
                    />
                  </TextTarget>
                ) : live ? (
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
                    element={mapped ? previewElement(element) : element}
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
});

function BlockTitle({ block, live }: { block: WorkBlock; live: boolean }) {
  const { setCursor, writeBlock } = useWorkspaceActions();
  const { cursor } = useWorkspaceFocus();
  const here = `block:${block.id}:title`;
  return (
    <EditableText
      active={cursor === here}
      activate={() => setCursor(here)}
      as="h2"
      className="font-display text-title font-medium tracking-tight text-ink [overflow-wrap:anywhere]"
      done={() => setCursor(null)}
      label={`Heading of ${block.title}`}
      live={live}
      onChange={(title) =>
        writeBlock({ ...block, title, titleIsDefault: title.trim() === "" })
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
