"use client";

import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";
import { Arrive } from "@/components/ui/arrive";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import type {
  AssetBlock,
  AssetElement,
  AssetImage,
  BrowseKind,
} from "@/lib/api/query";
import { blockCounts } from "@/lib/asset-block-heading";
import {
  assetHoldsNothing,
  coreBlockTitles,
  rendersOnThePage,
  splitAssetPageContent,
} from "@/lib/asset-page-content";
import { cn } from "@/lib/cn";
import {
  BLOCK_GRID_GAP_PX,
  elementTracks,
  ornamentPlacement,
  packBlockRows,
  pageFullness,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import { ContentsBar } from "./ContentsBar";
import { ElementBody } from "./ElementBody";
import {
  type ArtPlacement,
  EmptyPage,
  EmptyPageInvitation,
  QuietPageArt,
} from "./QuietPage";
import { useSuggestedWidths } from "./use-suggested-widths";
import { BlockTools } from "./workspace/BlockTools";
import { EditableElementSection } from "./workspace/EditableElement";
import { EditableText } from "./workspace/EditableText";
import { useWorkspace } from "./workspace/state";
import { useBlockDrag } from "./workspace/use-block-drag";

function returnToBlock(blockId: string) {
  const anchor = `block-${blockId}`;
  document.getElementById(anchor)?.scrollIntoView({ block: "start" });
  window.location.hash = anchor;
}

export function AssetBlocks({
  images,
  isOwner,
  kind,
  shellClassName,
}: {
  images: AssetImage[];
  isOwner: boolean;
  kind: BrowseKind;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const rowsNode = useRef<HTMLDivElement | null>(null);
  const known = useRef<Set<string> | null>(null);

  const blocks = workspace.blocks;
  const writing = isOwner && workspace.editing;
  const drag = useBlockDrag(workspace.arrangement.move);

  useEffect(() => {
    const seen = known.current;
    const fresh = seen ? blocks.find((block) => !seen.has(block.id)) : null;
    known.current = new Set(blocks.map((block) => block.id));
    if (fresh) returnToBlock(fresh.id);
  }, [blocks]);

  const { publicBlocks, modelContent: disclosedModelContent } = useMemo(
    () => splitAssetPageContent(blocks),
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
  const packable: Array<AssetBlock & { empty?: boolean }> = writing
    ? blocks
    : publicBlocks.filter(rendersOnThePage);
  const rows = packBlockRows(packable, { availableWidth });
  const fullness = pageFullness(rows);
  const invited = writing && assetHoldsNothing(blocks);
  const ornament = invited
    ? null
    : ornamentPlacement(rows, holdsCreatorPictures);
  const ornamentAtFoot =
    !invited &&
    rows.length > 0 &&
    !ornament &&
    !rows.some(holdsCreatorPictures);
  const contentsBlocks = useMemo(
    () => (writing ? blocks : publicBlocks.filter(rendersOnThePage)),
    [blocks, publicBlocks, writing],
  );
  const suggestedWidths = useSuggestedWidths({
    availableWidth,
    blocks,
    paused: !writing || workspace.cursor !== null,
    rows: rowsNode,
  });

  return (
    <>
      <ContentsBar
        blocks={contentsBlocks}
        shellClassName={shellClassName}
        writing={writing}
      />

      <div className={cn(shellClassName, "pt-10")}>
        {writing ? (
          <p className="mb-8 rounded-control bg-deep p-3 text-meta text-mute md:hidden">
            Block widths arrange the desktop page. On this screen every block
            fills the width, and no content is lost.
          </p>
        ) : null}
        {invited ? (
          <EmptyPageInvitation
            canAdd={workspace.addableBlocks.length > 0}
            coreBlocks={coreBlockTitles(blocks)}
            kind={kind}
          />
        ) : fullness === "empty" ? (
          <EmptyPage kind={kind} />
        ) : null}
        {rows.length === 0 ? null : (
          <div
            className="flex flex-col gap-14 md:gap-[5.5rem]"
            ref={setRowsRef}
            style={
              {
                "--block-grid-gap": `${BLOCK_GRID_GAP_PX}px`,
              } as CSSProperties
            }
          >
            {rows.map((row, rowIndex) => (
              <div
                className="grid grid-cols-1 items-start gap-14 md:grid-cols-12 md:gap-[var(--block-grid-gap)] md:gap-y-[5.5rem]"
                key={row.map((item) => item.block.id).join(":")}
              >
                {row.map(({ block, columns, startColumn }, place) => (
                  <Arrive
                    className="col-span-full min-w-0 md:[grid-column:var(--block-start)_/_span_var(--block-columns)]"
                    key={block.id}
                    place={writing ? 0 : place}
                    style={
                      {
                        "--block-columns": columns,
                        "--block-start": startColumn,
                      } as CSSProperties
                    }
                  >
                    <article
                      className={cn(
                        "group/block relative min-w-0 scroll-mt-[calc(var(--header-height)+5rem)] [container-name:block] [container-type:inline-size]",
                        writing &&
                          "after:pointer-events-none after:absolute after:-inset-x-5 after:-inset-y-4 after:rounded-plate after:opacity-0 after:ring-1 after:ring-accent/45 after:transition-opacity after:duration-200 after:content-[''] hover:after:opacity-100 focus-within:after:opacity-100 motion-reduce:after:transition-none",
                        drag.dragging === block.id && "opacity-45",
                        drag.over === block.id &&
                          "after:!opacity-100 after:!ring-2 after:!ring-accent",
                        writing && block.hidden && "bg-deep/60 px-5 pt-6 pb-7",
                      )}
                      data-block-id={block.id}
                      data-dragging={
                        drag.dragging === block.id ? true : undefined
                      }
                      data-hidden={writing && block.hidden ? true : undefined}
                      id={`block-${block.id}`}
                      {...(writing
                        ? drag.target(block.id, block.position)
                        : {})}
                    >
                      <header
                        className={cn(
                          "mb-5 flex flex-wrap items-end justify-between gap-x-6 gap-y-3.5",
                          writing && block.hidden ? "opacity-50" : null,
                        )}
                      >
                        <div className="flex min-w-0 flex-1 basis-45 flex-wrap items-baseline gap-x-2.5 gap-y-1">
                          <BlockTitle block={block} />
                          {writing && block.required ? (
                            <span className="shrink-0 rounded-control bg-deep px-2 py-1 text-label text-mute">
                              {block.hideable ? "Required" : "Always shown"}
                            </span>
                          ) : null}
                          <BlockCounts elements={block.elements} />
                        </div>
                        {writing ? (
                          <BlockTools
                            block={block}
                            grip={drag.grip(block.id)}
                            position={block.position}
                            suggestedWidth={suggestedWidths[block.id]}
                            total={blocks.length}
                          />
                        ) : null}
                      </header>
                      {writing && block.hidden ? (
                        <div className="-mt-1 mb-5 flex flex-col items-stretch justify-between gap-3 rounded-control bg-plane p-3 text-meta text-mute sm:flex-row sm:items-center">
                          <span>
                            Hidden from readers. This content is still included
                            in downloads.
                          </span>
                          <button
                            className="min-h-11 shrink-0 rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45"
                            onClick={() =>
                              workspace.arrangement.setHidden(block.id, false)
                            }
                            type="button"
                          >
                            Show block
                          </button>
                        </div>
                      ) : null}
                      <div
                        className={cn(
                          "grid gap-x-8 gap-y-7 [grid-template-columns:var(--element-tracks,minmax(0,1fr))] max-md:![grid-template-columns:minmax(0,1fr)]",
                          writing && block.hidden ? "opacity-50" : null,
                        )}
                        data-block-content
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
                            {writing ? (
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
                  </Arrive>
                ))}
                {ornament?.row === rowIndex ? (
                  <Ornament
                    barren={fullness === "barren"}
                    placement="inRow"
                    style={
                      {
                        "--block-columns": ornament.columns,
                        "--block-start": ornament.startColumn,
                      } as CSSProperties
                    }
                  />
                ) : null}
              </div>
            ))}
            {ornamentAtFoot ? (
              <Ornament barren={fullness === "barren"} placement="atFoot" />
            ) : null}
          </div>
        )}
        {modelContent.length > 0 ? (
          <MorphingDisclosure
            className="mt-section rounded-plate bg-deep/70 px-5 py-3"
            summary="Model instructions"
            trailing={
              <span className="font-ui text-meta text-mute">
                What the creator tells the model, kept out of the reading order
              </span>
            }
          >
            <div className="grid gap-8 pt-5 pb-2">
              {modelContent.map(({ element }) => (
                <ElementBody
                  element={element}
                  images={images}
                  isOwner={false}
                  key={element.id}
                />
              ))}
            </div>
          </MorphingDisclosure>
        ) : null}
      </div>
    </>
  );
}

function BlockTitle({ block }: { block: AssetBlock }) {
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
      live={workspace.editing}
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

function Ornament({
  barren,
  placement,
  style,
}: {
  barren: boolean;
  placement: Extract<ArtPlacement, "inRow" | "atFoot">;
  style?: CSSProperties;
}) {
  return <QuietPageArt compact={!barren} placement={placement} style={style} />;
}

function BlockCounts({ elements }: { elements: AssetElement[] }) {
  const counts = blockCounts(elements);
  if (!counts) return null;
  return <p className="basis-full text-label text-mute">{counts}</p>;
}

function holdsCreatorPictures(row: readonly { block: AssetBlock }[]): boolean {
  return row.some(({ block }) =>
    block.elements.some(
      (element) => element.type === "image_set" && !element.isEmpty,
    ),
  );
}
