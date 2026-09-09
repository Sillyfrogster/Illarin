"use client";

import { ChevronDown } from "lucide-react";
import { useRouter } from "next/navigation";
import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  type AddableBlock,
  type AssetBlock,
  type AssetElement,
  type AssetImage,
  addAssetBlock,
  arrangeAssetBlocks,
  type BrowseKind,
  type ElementType,
  moveAssetBlockContent,
  removeAssetBlock,
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
import { pageWashVariables } from "@/lib/quiet-page-art";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import { useWorkingCopy } from "@/lib/working-copy";
import { AddBlockTray } from "./AddBlockTray";
import {
  ArrangeBlocks,
  moveContentDestinations,
  RemoveBlockDialog,
} from "./ArrangeBlocks";
import { BlockTools } from "./BlockTools";
import { ContentsBar } from "./ContentsBar";
import { ElementBody } from "./ElementBody";
import { ElementReader } from "./ElementReader";
import {
  type ArtPlacement,
  EmptyPage,
  EmptyPageInvitation,
  QuietPageArt,
} from "./QuietPage";
import { useSuggestedWidths } from "./use-suggested-widths";
import { EditableElementSection } from "./workspace/EditableElement";
import { EditableText } from "./workspace/EditableText";
import { useWorkspace } from "./workspace/state";

function returnToBlock(blockId: string) {
  const anchor = `block-${blockId}`;
  document.getElementById(anchor)?.scrollIntoView({ block: "start" });
  window.location.hash = anchor;
}

/** The asset's content. An owner also sees the blocks they have yet to fill. */
export function AssetBlocks({
  addableBlocks,
  assetId,
  images,
  isOwner,
  kind,
  shellClassName,
}: {
  addableBlocks: AddableBlock[];
  assetId: string;
  images: AssetImage[];
  isOwner: boolean;
  kind: BrowseKind;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const candidate = useWorkingCopy();
  const router = useRouter();
  const [arrangementMessage, setArrangementMessage] = useState("");
  const [arranging, setArranging] = useState(false);
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState<AssetBlock | null>(null);
  const [blockActionPending, setBlockActionPending] = useState(false);
  const [added, setAdded] = useState<string | null>(null);
  const [reading, setReading] = useState<{
    blockId: string;
    element: AssetElement;
  } | null>(null);
  const rowsNode = useRef<HTMLDivElement | null>(null);

  const blocks = workspace.blocks;
  const writing = isOwner && workspace.editing;

  useEffect(() => {
    if (!added) return;
    returnToBlock(added);
    setAdded(null);
  }, [added]);

  useEffect(() => {
    if (!adding) return;
    window.requestAnimationFrame(() => {
      document
        .getElementById("add-block-tray")
        ?.scrollIntoView({ block: "start" });
    });
  }, [adding]);

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
  /** An owner filling in an empty page is invited once, not block by block. */
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
    paused: !writing || arranging || workspace.cursor !== null,
    rows: rowsNode,
  });

  function dismissReader() {
    const elementId = reading?.element.id;
    setReading(null);
    if (!elementId) return;
    window.requestAnimationFrame(() => {
      document.getElementById(`read-${elementId}`)?.focus({
        preventScroll: true,
      });
    });
  }

  async function runBlockAction(action: () => Promise<void>) {
    if (blockActionPending) return;
    setBlockActionPending(true);
    setArrangementMessage("");
    try {
      await action();
      router.refresh();
    } catch (error) {
      setArrangementMessage(
        error instanceof Error
          ? error.message
          : "The block could not be changed. Try again.",
      );
    } finally {
      setBlockActionPending(false);
    }
  }

  function addBlock(definition: string, elementType: ElementType) {
    void runBlockAction(async () => {
      const block = await addAssetBlock(
        candidate,
        assetId,
        definition,
        elementType,
      );
      workspace.editBlockList((list) => [...list, block]);
      setArranging(false);
      setAdding(false);
      setAdded(block.id);
    });
  }

  function hideBlock(blockId: string) {
    void runBlockAction(async () => {
      const saved = await arrangeAssetBlocks(candidate, assetId, {
        blocks: blocks.map((block) => ({
          hidden: block.id === blockId ? true : block.hidden,
          id: block.id,
          width: block.width,
        })),
      });
      workspace.applyServerBlocks(saved);
    });
  }

  function showBlock(blockId: string) {
    void runBlockAction(async () => {
      const saved = await arrangeAssetBlocks(candidate, assetId, {
        blocks: blocks.map((block) => ({
          hidden: block.id === blockId ? false : block.hidden,
          id: block.id,
          width: block.width,
        })),
      });
      workspace.applyServerBlocks(saved);
    });
  }

  return (
    <>
      <ContentsBar
        adding={adding}
        arranging={arranging}
        blocks={contentsBlocks}
        canAdd={addableBlocks.length > 0}
        onToggleAdd={() => {
          setArranging(false);
          setAdding((current) => !current);
        }}
        onToggleArrange={() => {
          setAdding(false);
          setArranging((current) => !current);
        }}
        shellClassName={shellClassName}
        writing={writing}
      />

      <div className={cn(shellClassName, "pt-10")}>
        {arranging && writing ? (
          <ArrangeBlocks
            assetId={assetId}
            blocks={blocks}
            onChange={workspace.applyServerBlocks}
            onClose={() => setArranging(false)}
            suggestedWidths={suggestedWidths}
          />
        ) : (
          <>
            {writing ? (
              <p className="mb-8 rounded-control bg-deep p-3 text-meta text-mute md:hidden">
                Block widths arrange the desktop page. On this screen every
                block fills the width, and no content is lost.
              </p>
            ) : null}
            {arrangementMessage ? (
              <p
                className="mb-5 rounded-control bg-stop-wash p-3 text-meta text-ink"
                role="alert"
              >
                {arrangementMessage}
              </p>
            ) : null}
            {invited ? (
              <EmptyPageInvitation
                canAdd={addableBlocks.length > 0}
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
                    {row.map(({ block, columns, startColumn }) => (
                      <article
                        className={cn(
                          "group/block relative min-w-0 scroll-mt-[calc(var(--header-height)+5rem)] [container-name:block] [container-type:inline-size]",
                          "col-span-full md:[grid-column:var(--block-start)_/_span_var(--block-columns)]",
                          writing &&
                            "after:pointer-events-none after:absolute after:-inset-x-5 after:-inset-y-4 after:rounded-plate after:opacity-0 after:ring-1 after:ring-accent/45 after:transition-opacity after:duration-200 after:content-[''] hover:after:opacity-100 focus-within:after:opacity-100 motion-reduce:after:transition-none",
                          writing &&
                            block.hidden &&
                            "bg-deep/60 px-5 pt-6 pb-7",
                        )}
                        data-block-id={block.id}
                        data-hidden={writing && block.hidden ? true : undefined}
                        id={`block-${block.id}`}
                        key={block.id}
                        style={
                          {
                            "--block-columns": columns,
                            "--block-start": startColumn,
                          } as CSSProperties
                        }
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
                              onHide={() => hideBlock(block.id)}
                              onIssue={setArrangementMessage}
                              onRemove={() => setRemoving(block)}
                              suggestedWidth={suggestedWidths[block.id]}
                            />
                          ) : null}
                        </header>
                        {writing && block.hidden ? (
                          <div className="-mt-1 mb-5 flex flex-col items-stretch justify-between gap-3 rounded-control bg-plane p-3 text-meta text-mute sm:flex-row sm:items-center">
                            <span>
                              Hidden from the public page. Everything in it is
                              kept, and it still travels in every download.
                            </span>
                            <button
                              className="min-h-11 shrink-0 rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45"
                              onClick={() => showBlock(block.id)}
                              type="button"
                            >
                              Show it again
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
                                  onReadMore={() =>
                                    setReading({ blockId: block.id, element })
                                  }
                                />
                              ) : (
                                <ElementBody
                                  blockElements={block.elements.length}
                                  blockTitle={block.title}
                                  element={element}
                                  images={images}
                                  isOwner={false}
                                  markEmpty={!invited}
                                  onReadMore={() =>
                                    setReading({ blockId: block.id, element })
                                  }
                                />
                              )}
                              {reading?.blockId === block.id &&
                              reading.element.id === element.id ? (
                                <ElementReader
                                  element={reading.element}
                                  images={images}
                                  onDismiss={dismissReader}
                                />
                              ) : null}
                            </div>
                          ))}
                        </div>
                      </article>
                    ))}
                    {ornament?.row === rowIndex ? (
                      <Ornament
                        barren={fullness === "barren"}
                        kind={kind}
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
                  <Ornament
                    barren={fullness === "barren"}
                    kind={kind}
                    placement="atFoot"
                  />
                ) : null}
              </div>
            )}
            {modelContent.length > 0 ? (
              <details className="group mt-section rounded-plate bg-deep/70">
                <summary className="flex min-h-16 cursor-pointer list-none flex-wrap items-center justify-between gap-x-4 gap-y-1 p-5 outline-offset-3 [&::-webkit-details-marker]:hidden">
                  <span className="text-ui font-medium text-ink">
                    Model-facing content
                  </span>
                  <span className="text-meta text-mute">
                    System prompt and post-history instructions
                    <ChevronDown
                      aria-hidden="true"
                      className="ml-3 inline size-4 align-middle transition-transform duration-200 group-open:rotate-180 motion-reduce:transition-none"
                    />
                  </span>
                </summary>
                <div className="grid gap-7 px-5 pb-5">
                  {modelContent.map(({ block, element }) => (
                    <div key={element.id}>
                      <ElementBody
                        element={element}
                        images={images}
                        isOwner={false}
                        onReadMore={() =>
                          setReading({ blockId: block.id, element })
                        }
                      />
                      {reading?.blockId === block.id &&
                      reading.element.id === element.id ? (
                        <ElementReader
                          element={reading.element}
                          images={images}
                          onDismiss={dismissReader}
                        />
                      ) : null}
                    </div>
                  ))}
                </div>
              </details>
            ) : null}
            {writing && adding ? (
              <AddBlockTray
                addable={addableBlocks}
                blocks={blocks}
                onAdd={addBlock}
                onClose={() => setAdding(false)}
                pending={blockActionPending}
              />
            ) : null}
          </>
        )}
      </div>
      {removing ? (
        <RemoveBlockDialog
          block={removing}
          destinations={moveContentDestinations(removing, blocks)}
          error={arrangementMessage}
          onCancel={() => setRemoving(null)}
          onHide={() =>
            runBlockAction(async () => {
              const saved = await arrangeAssetBlocks(candidate, assetId, {
                blocks: blocks.map((block) => ({
                  hidden: block.id === removing.id ? true : block.hidden,
                  id: block.id,
                  width: block.width,
                })),
              });
              workspace.applyServerBlocks(saved);
              setRemoving(null);
            })
          }
          onMove={(destinationBlockId) =>
            runBlockAction(async () => {
              const saved = await moveAssetBlockContent(
                candidate,
                assetId,
                removing.id,
                destinationBlockId,
              );
              workspace.applyServerBlocks(saved);
              setRemoving(null);
            })
          }
          onRemove={() =>
            runBlockAction(async () => {
              await removeAssetBlock(candidate, assetId, removing.id);
              workspace.editBlockList((list) =>
                list
                  .filter((block) => block.id !== removing.id)
                  .map((block, position) => ({ ...block, position })),
              );
              setRemoving(null);
            })
          }
          pending={blockActionPending}
        />
      ) : null}
    </>
  );
}

/** A block heading a creator writes over, and a save restores where they clear it. */
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

/**
 * The artwork that takes what a row leaves. A page with plenty on it gets the
 * shared wash. A page with one row gets the kind's own piece, because there
 * the artwork is the composition rather than a hint of one.
 */
function Ornament({
  barren,
  kind,
  placement,
  style,
}: {
  barren: boolean;
  kind: BrowseKind;
  placement: Extract<ArtPlacement, "inRow" | "atFoot">;
  style?: CSSProperties;
}) {
  if (barren) {
    return <QuietPageArt kind={kind} placement={placement} style={style} />;
  }
  return (
    <div
      aria-hidden="true"
      className={cn(
        "relative",
        placement === "inRow"
          ? "min-h-52 md:min-h-60 md:[grid-column:var(--block-start)_/_span_var(--block-columns)]"
          : "h-52 md:mt-10 md:h-70",
        "[--art-bleed:max(72px,(100vw-var(--shell))/2+var(--gutter))]",
        "before:absolute before:-z-1 before:bg-[image:var(--ornament-light)] before:bg-cover before:bg-[position:center_32%] before:bg-no-repeat before:opacity-50 before:content-['']",
        "before:[mask-composite:intersect] before:[mask-image:linear-gradient(to_right,transparent,#000_52%),linear-gradient(to_bottom,transparent,#000_30%,#000_62%,transparent)]",
        "dark:before:bg-[image:var(--ornament-dark)] dark:before:opacity-[0.78]",
        placement === "inRow"
          ? "before:inset-y-0 before:left-[10%] before:w-[calc(90%+var(--gutter))] md:before:-inset-y-12 md:before:left-0 md:before:w-[calc(100%+var(--art-bleed))]"
          : "before:inset-y-0 before:left-[10%] before:w-[calc(90%+var(--gutter))] md:before:left-[46%] md:before:w-[calc(54%+var(--art-bleed))]",
      )}
      data-measurement-ignore
      style={{ ...pageWashVariables(), ...style }}
    />
  );
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
