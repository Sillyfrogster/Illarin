"use client";

import { motion } from "framer-motion";
import { EyeOff, Layers, Plus } from "lucide-react";
import Image from "next/image";
import darkArt from "@/assets/art/full/illarin-detail-page-art-dark-v2.webp";
import lightArt from "@/assets/art/full/illarin-detail-page-art-light-v2.webp";
import {
  elementTracks,
  NARROW_BLOCK_GRID_PX,
  packBlockRows,
  WIDTH_LABELS,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import type { Asset, Block, Element } from "./data";
import type { Focus } from "./session";
import { cn, Eyebrow } from "./ui";

export const DETAILS_REGION = "region:details";
export const elementRegion = (id: string) => `region:${id}`;

function hasContent(element: Element) {
  return Boolean(
    element.text.trim() || element.items.some((item) => item.text.trim()),
  );
}

/** An editable region's outline, a div rather than a button so it can hold buttons */
function Region({
  id,
  live,
  active,
  changed,
  empty,
  children,
}: {
  id: string;
  live: boolean;
  active: boolean;
  changed?: boolean;
  empty?: boolean;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      layoutId={live ? id : undefined}
      data-live={live ? "true" : "false"}
      data-empty={live && empty ? "true" : undefined}
      className={cn("w-refract ws:min-w-0", active && "ws:opacity-0")}
    >
      {changed && live && (
        <span
          aria-hidden="true"
          className="ws:absolute ws:-top-2.5 ws:-left-3.5 ws:size-2 ws:rounded-full ws:bg-amber"
        />
      )}
      {children}
    </motion.div>
  );
}

function RegionButton({
  label,
  live,
  onOpen,
  className,
  children,
}: {
  label: string;
  live: boolean;
  onOpen: () => void;
  className?: string;
  children: React.ReactNode;
}) {
  if (!live)
    return <div className={cn("ws:min-w-0", className)}>{children}</div>;
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`Write ${label}`}
      className={cn(
        "ws:block ws:w-full ws:min-w-0 ws:cursor-text ws:text-left",
        className,
      )}
    >
      {children}
    </button>
  );
}

function ItemList({
  element,
  live,
  open,
}: {
  element: Element;
  live: boolean;
  open: (itemId: string) => void;
}) {
  const items = live
    ? element.items
    : element.items.filter((item) => item.text.trim());
  const conversation = element.type === "dialogue_sample";
  if (!items.length && !live)
    return (
      <p className="ws:text-sm ws:text-mute ws:italic">Nothing here yet.</p>
    );
  return (
    <ul
      className={cn(
        "ws:-mx-3 ws:min-w-0",
        element.type !== "dialogue_sample" && "w-items",
      )}
    >
      {items.map((item, index) => {
        const body = (
          <>
            <span className="ws:flex ws:min-w-0 ws:items-baseline ws:gap-3">
              <span className="ws:min-w-0 ws:flex-1 ws:font-semibold ws:tracking-tight ws:wrap-anywhere">
                {item.name || (
                  <span className="ws:text-mute ws:italic">Untitled</span>
                )}
              </span>
              {item.keys && (
                <span className="ws:hidden ws:shrink-0 ws:text-xs ws:text-mute ws:sm:inline">
                  {item.keys.split(",")[0].trim()}
                </span>
              )}
              {item.enabled === false && (
                <span className="ws:shrink-0 ws:text-xs ws:text-mute">off</span>
              )}
            </span>
            <span
              className={cn(
                "w-writing ws:mt-1.5 ws:block ws:max-w-[62ch] ws:text-[1.0625rem] ws:leading-7 ws:text-mute",
                conversation ? "ws:line-clamp-none" : "ws:line-clamp-2",
              )}
            >
              {item.text.trim() || (live ? "Write this one." : "") || " "}
            </span>
          </>
        );
        return (
          <li key={item.id}>
            {live ? (
              <button
                type="button"
                onClick={() => open(item.id)}
                className="ws:block ws:w-full ws:rounded-xl ws:px-3 ws:py-2.5 ws:text-left ws:transition ws:hover:bg-ink/6 ws:motion-reduce:transition-none"
              >
                {body}
              </button>
            ) : (
              <div
                className={cn(
                  "ws:px-3 ws:py-2.5",
                  index > 0 && "ws:shadow-[inset_0_1px_0_var(--w-hairline)]",
                )}
              >
                {body}
              </div>
            )}
          </li>
        );
      })}
      {live && (
        <li className="ws:col-span-full">
          <span className="ws:flex ws:items-center ws:gap-2 ws:px-3 ws:pt-2 ws:text-sm ws:text-mute">
            <Plus className="ws:size-3.5" />
            {element.items.length}{" "}
            {element.items.length === 1 ? "entry" : "entries"} · open to add or
            reorder
          </span>
        </li>
      )}
    </ul>
  );
}

function BlockSection({
  block,
  live,
  focus,
  changed,
  open,
  arrange,
  width,
}: {
  block: Block;
  live: boolean;
  focus: Focus;
  changed: Set<string>;
  open: (elementId: string, itemId?: string) => void;
  arrange: () => void;
  width?: number;
}) {
  const elements = live ? block.elements : block.elements.filter(hasContent);
  if (!elements.length) return null;
  const narrow = width !== undefined && width <= NARROW_BLOCK_GRID_PX;
  return (
    <section className="ws:min-w-0">
      <header className="ws:mb-6 ws:flex ws:items-baseline ws:gap-4">
        <h2 className="ws:min-w-0 ws:font-display ws:text-[1.75rem] ws:leading-tight ws:font-medium ws:wrap-anywhere ws:md:text-4xl">
          {block.title || "Untitled block"}
        </h2>
        {live && (
          <button
            type="button"
            onClick={arrange}
            className="ws:inline-flex ws:shrink-0 ws:items-center ws:gap-1.5 ws:rounded-full ws:px-2.5 ws:py-1 ws:text-xs ws:font-semibold ws:text-mute ws:transition ws:hover:bg-ink/8 ws:hover:text-ink ws:motion-reduce:transition-none"
          >
            {block.hidden ? (
              <EyeOff className="ws:size-3.5" />
            ) : (
              <Layers className="ws:size-3.5" />
            )}
            {block.hidden ? "Hidden" : WIDTH_LABELS[block.width]}
          </button>
        )}
      </header>
      <div
        className="ws:grid ws:gap-x-8 ws:gap-y-10"
        style={{
          gridTemplateColumns: narrow
            ? "minmax(0,1fr)"
            : elementTracks(block.layout, elements.length),
        }}
      >
        {elements.map((element) => {
          const id = elementRegion(element.id);
          const active =
            focus?.type === "element" && focus.elementId === element.id;
          const empty = !hasContent(element);
          return (
            <Region
              key={element.id}
              id={id}
              live={live}
              active={active}
              changed={changed.has(element.id)}
              empty={empty}
            >
              {(elements.length > 1 || element.type !== "prose") && (
                <Eyebrow className="ws:mb-3">{element.label}</Eyebrow>
              )}
              {element.type === "prose" ? (
                <RegionButton
                  label={element.label}
                  live={live}
                  onOpen={() => open(element.id)}
                >
                  <p
                    className={cn(
                      "w-writing ws:max-w-[64ch] ws:whitespace-pre-wrap ws:wrap-anywhere",
                      empty && "ws:text-mute ws:italic",
                    )}
                  >
                    {element.text.trim() ||
                      (live ? `Write the ${element.label.toLowerCase()}.` : "")}
                  </p>
                </RegionButton>
              ) : (
                <ItemList
                  element={element}
                  live={live}
                  open={(itemId) => open(element.id, itemId)}
                />
              )}
            </Region>
          );
        })}
      </div>
    </section>
  );
}

export function AssetPage({
  asset,
  live,
  focus,
  theme,
  changed,
  open,
  openDetails,
  arrange,
  action,
}: {
  asset: Asset;
  live: boolean;
  focus: Focus;
  theme: "light" | "dark";
  changed: Set<string>;
  open: (elementId: string, itemId?: string) => void;
  openDetails: () => void;
  arrange: () => void;
  action: React.ReactNode;
}) {
  const [ref, width] = useMeasuredWidth<HTMLDivElement>();
  const rows = packBlockRows(
    live ? asset.blocks : asset.blocks.filter((b) => !b.hidden),
    { availableWidth: width },
  );
  const detailsActive = focus?.type === "details";
  return (
    <div className="ws:w-full ws:overflow-x-clip">
      <div className="ws:relative ws:mx-auto ws:max-w-[86rem] ws:px-5 ws:pt-10 ws:pb-16 ws:md:px-10 ws:md:pt-16 ws:md:pb-24">
        <Image
          src={theme === "light" ? lightArt : darkArt}
          alt=""
          priority
          sizes="(max-width: 900px) 100vw, 900px"
          style={{
            maskImage:
              "linear-gradient(to left, black 6%, transparent 82%), linear-gradient(to top, transparent 2%, black 34%)",
            maskComposite: "intersect",
            WebkitMaskComposite: "source-in",
          }}
          className="ws:pointer-events-none ws:absolute ws:top-0 ws:right-[-10%] ws:h-full ws:w-[72%] ws:object-cover ws:opacity-50 ws:max-md:opacity-20"
        />
        <div className="ws:relative ws:max-w-3xl">
          <Region
            id={DETAILS_REGION}
            live={live}
            active={detailsActive}
            changed={
              changed.has("name") ||
              changed.has("blurb") ||
              changed.has("version") ||
              changed.has("nsfw")
            }
          >
            <RegionButton
              label="the asset details"
              live={live}
              onOpen={openDetails}
            >
              <Eyebrow>
                {asset.kind} · {asset.version || "No version label"}
                {asset.nsfw === "yes" && " · Adult content"}
                {asset.nsfw === "unanswered" && " · Adult content unanswered"}
              </Eyebrow>
              <h1 className="ws:mt-5 ws:font-display ws:text-[clamp(2.75rem,7vw,5.25rem)] ws:leading-[1.03] ws:font-medium ws:tracking-[-0.015em] ws:wrap-anywhere">
                {asset.name || (
                  <span className="ws:text-mute ws:italic">Untitled asset</span>
                )}
              </h1>
              <p className="w-writing ws:mt-6 ws:max-w-2xl ws:text-[1.3125rem] ws:leading-9 ws:text-mute ws:wrap-anywhere">
                {asset.blurb || (live ? "Add a short blurb." : "")}
              </p>
            </RegionButton>
          </Region>
          <div className="ws:mt-10 ws:flex ws:flex-wrap ws:items-center ws:gap-3">
            {action}
          </div>
        </div>
      </div>

      <div
        ref={ref}
        className="ws:mx-auto ws:max-w-[86rem] ws:space-y-16 ws:px-5 ws:pb-40 ws:md:space-y-20 ws:md:px-10"
      >
        {rows.map((row) => (
          <div
            key={row[0].block.id}
            className="ws:-mx-4 ws:grid ws:grid-cols-12 ws:gap-y-16"
          >
            {row.map(({ block, columns, startColumn }) => (
              <div
                key={block.id}
                style={{ gridColumn: `${startColumn} / span ${columns}` }}
                className={cn(
                  "ws:min-w-0 ws:px-4",
                  live && block.hidden && "ws:opacity-55",
                )}
              >
                <BlockSection
                  block={block}
                  live={live}
                  focus={focus}
                  changed={changed}
                  open={open}
                  arrange={arrange}
                  width={width}
                />
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
