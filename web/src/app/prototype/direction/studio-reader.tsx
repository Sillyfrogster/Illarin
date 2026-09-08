"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import {
  ArrowDown,
  ArrowLeft,
  Check,
  ChevronDown,
  Download,
  ExternalLink,
  History,
  Pencil,
  Send,
  X,
} from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import {
  BLOCK_GRID_GAP_PX,
  elementTracks,
  packBlockRows,
} from "@/lib/page-arrangement";
import { useMeasuredWidth } from "@/lib/use-measured-width";
import type { Asset, Block } from "./assets";
import { KIND_LABEL } from "./assets";
import { Button } from "./components/button";
import { PerspectiveCarousel } from "./components/perspective-carousel";
import { Popover, PopoverContent, PopoverTrigger } from "./components/popover";
import { ElementView } from "./element";
import { cn } from "./ui";

function GetAsset({ asset, failOnce }: { asset: Asset; failOnce: boolean }) {
  const [selected, setSelected] = useState(
    asset.formats.findIndex((format) => format.recommended),
  );
  const [destination, setDestination] = useState("download");
  const [status, setStatus] = useState("");
  const [failed, setFailed] = useState(false);
  const [attempted, setAttempted] = useState(false);
  const format = asset.formats[Math.max(0, selected)];
  return (
    <Popover onOpenChange={() => setStatus("")}>
      <PopoverTrigger asChild>
        <Button className="vd:min-w-44 vd:justify-between">
          Get {KIND_LABEL[asset.kind].toLowerCase()}
          <ArrowDown />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={12} aria-label="Get this asset">
        <h2 className="vd:text-section vd:font-medium">Take it with you</h2>
        <p className="vd:mt-1 vd:text-meta vd:text-mute">
          Choose a format and where it goes.
        </p>
        <fieldset className="vd:mt-5 vd:space-y-1">
          <legend className="vd:mb-2 vd:text-meta vd:font-medium">
            Format
          </legend>
          {asset.formats.map((item, index) => (
            <label
              key={item.label}
              className={cn(
                "vd:flex vd:min-h-12 vd:cursor-pointer vd:items-center vd:gap-3 vd:rounded-control vd:px-3 vd:py-2",
                selected === index && "vd:bg-accent-wash",
              )}
            >
              <input
                type="radio"
                name="format"
                checked={selected === index}
                onChange={() => {
                  setSelected(index);
                  setStatus("");
                }}
                disabled={item.verdict === "blocked"}
                className="vd:size-4 vd:accent-accent"
              />
              <span className="vd:min-w-0 vd:text-ui">
                {item.label}
                {item.recommended && (
                  <span className="vd:block vd:text-meta vd:text-mute">
                    Recommended
                  </span>
                )}
              </span>
            </label>
          ))}
        </fieldset>
        {format?.drops ? (
          <p className="vd:mt-3 vd:text-meta vd:text-stop">
            Not included: {format.drops.join(", ")}.
          </p>
        ) : (
          <p className="vd:mt-3 vd:flex vd:items-center vd:gap-2 vd:text-meta vd:text-mute">
            <Check className="vd:size-4" />
            Includes all supported content
          </p>
        )}
        <label
          className="vd:mt-5 vd:block vd:text-meta vd:font-medium"
          htmlFor="destination"
        >
          Send to
        </label>
        <select
          id="destination"
          value={destination}
          onChange={(event) => {
            setDestination(event.target.value);
            setStatus("");
          }}
          className="vd:mt-2 vd:h-11 vd:w-full vd:rounded-control vd:bg-deep vd:px-3 vd:text-ui"
        >
          <option value="download">Download a file</option>
          <option value="linked">My Lumiverse · linked app</option>
        </select>
        <Button
          className="vd:mt-4 vd:w-full"
          disabled={!format || format.verdict === "blocked"}
          onClick={() => {
            if (failOnce && !attempted) {
              setAttempted(true);
              setFailed(true);
              setStatus("");
              return;
            }
            setFailed(false);
            setStatus(
              destination === "download"
                ? "Download preview complete. No file was created."
                : "Delivery preview complete. No app was contacted.",
            );
          }}
        >
          {destination === "download" ? <Download /> : <Send />}
          {failed
            ? "Try again"
            : destination === "download"
              ? "Download"
              : "Send to Lumiverse"}
        </Button>
        {failed && (
          <p role="alert" className="vd:mt-3 vd:text-meta vd:text-stop">
            Could not complete the request. Your choices are saved. Try again.
          </p>
        )}
        <output className="vd:mt-3 vd:block vd:text-meta vd:text-mute">
          {status ||
            "Synthetic preview. Downloads and linked delivery are simulated."}
        </output>
      </PopoverContent>
    </Popover>
  );
}

function ReaderBlock({ block }: { block: Block }) {
  const [ref, width] = useMeasuredWidth<HTMLDivElement>();
  return (
    <section
      id={`block-${block.id}`}
      data-block={block.id}
      className="vd:min-w-0 vd:scroll-mt-28"
    >
      <h2 className="vd:mb-5 vd:text-title vd:font-medium vd:tracking-tight">
        {block.title}
      </h2>
      <div
        ref={ref}
        className="vd:grid vd:gap-8"
        style={{
          gridTemplateColumns:
            (width ?? 0) < 540
              ? "minmax(0,1fr)"
              : elementTracks(block.layout, block.elements.length),
        }}
      >
        {block.elements.map((element) => (
          <div key={element.id} className="vd:min-w-0">
            {element.type === "image_set" ? (
              <PerspectiveCarousel images={element.images} />
            ) : (
              <ElementView
                element={element}
                direction="vitrine"
                labelled={block.elements.length > 1}
              />
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

export function StudioReader({
  asset,
  canEdit = true,
  failOnce = false,
}: {
  asset: Asset;
  canEdit?: boolean;
  failOnce?: boolean;
}) {
  const [ref, width] = useMeasuredWidth<HTMLDivElement>();
  const [intro, setIntro] = useState(asset.blurb ?? "");
  const [draft, setDraft] = useState(intro);
  const [editing, setEditing] = useState(false);
  const [saved, setSaved] = useState(false);
  const [history, setHistory] = useState(false);
  const reduced = useReducedMotion();
  const rows = packBlockRows(asset.blocks, { availableWidth: width });
  return (
    <main
      id="main-content"
      className="vd:px-5 vd:pb-16 vd:md:px-10 vd:xl:px-14"
    >
      <a
        href="/browse"
        className="vd:mt-6 vd:inline-flex vd:min-h-11 vd:items-center vd:gap-2 vd:text-meta vd:text-mute vd:hover:text-ink"
      >
        <ArrowLeft className="vd:size-4" />
        Back to the collection
      </a>
      <div
        className={cn(
          "vd:grid vd:items-center vd:gap-7 vd:py-8 vd:lg:gap-10 vd:lg:py-12",
          asset.cover
            ? "vd:md:grid-cols-[1fr_1fr] vd:lg:grid-cols-[1fr_minmax(260px,0.9fr)_1fr]"
            : "vd:md:grid-cols-[1.2fr_1fr]",
        )}
      >
        <div className="vd:min-w-0">
          <h1 className="vd:max-w-[15ch] vd:font-display vd:text-hero vd:font-medium vd:tracking-[-0.035em] vd:break-words">
            {asset.name}
          </h1>
          <p className="vd:mt-5 vd:flex vd:items-center vd:gap-2 vd:text-ui vd:text-mute">
            <span className="vd:size-2 vd:rounded-full vd:bg-accent" />
            {KIND_LABEL[asset.kind]}
            <span aria-hidden="true">·</span>v{asset.assetVersion}
          </p>
          <div className="vd:mt-8 vd:flex vd:items-center vd:gap-3">
            <span
              aria-hidden="true"
              className="vd:flex vd:size-11 vd:items-center vd:justify-center vd:rounded-full vd:bg-accent-wash vd:text-accent"
            >
              {asset.creditedAuthor.slice(0, 1)}
            </span>
            <div>
              <p className="vd:text-ui">{asset.creditedAuthor}</p>
              <p className="vd:text-meta vd:text-mute">
                {asset.creator} · {asset.shared}
              </p>
            </div>
          </div>
        </div>
        {asset.cover && (
          <motion.figure
            initial={false}
            whileHover={reduced ? undefined : { rotate: -1, y: -5 }}
            transition={{ type: "spring", stiffness: 180, damping: 22 }}
            className="vd:relative vd:mx-auto vd:w-full vd:max-w-[350px] vd:md:row-span-2 vd:lg:row-span-1"
          >
            <a
              href={asset.cover.src}
              target="_blank"
              rel="noreferrer"
              aria-label="View full cover"
              className="vd:group vd:block vd:rounded-plate"
            >
              <Image
                src={asset.cover}
                alt={`Cover of ${asset.name}`}
                priority
                sizes="(max-width: 767px) 88vw, 350px"
                className="vd:h-auto vd:w-full vd:rounded-plate vd:shadow-[0_20px_45px_-20px_rgb(0_0_0/0.5)]"
              />
              <span className="vd:absolute vd:right-3 vd:bottom-3 vd:flex vd:size-11 vd:items-center vd:justify-center vd:rounded-full vd:bg-field vd:text-ink vd:opacity-100 vd:transition-opacity vd:lg:opacity-0 vd:lg:group-hover:opacity-100 vd:lg:group-focus-visible:opacity-100">
                <ExternalLink className="vd:size-4" />
              </span>
            </a>
          </motion.figure>
        )}
        <div className="vd:min-w-0 vd:md:col-start-1 vd:lg:col-start-auto">
          <p className="vd:max-w-[36ch] vd:font-prose vd:text-lede vd:text-ink">
            {intro || "The creator has not added an introduction."}
          </p>
          {asset.tags.length > 0 && (
            <p className="vd:mt-5 vd:max-w-[36ch] vd:text-meta vd:leading-7 vd:text-mute">
              {asset.tags.join(" / ")}
            </p>
          )}
          <div className="vd:mt-7 vd:flex vd:flex-wrap vd:items-center vd:gap-3">
            <GetAsset asset={asset} failOnce={failOnce} />
            <Button
              variant="ghost"
              size="icon"
              aria-label="Show update history"
              onClick={() => setHistory(!history)}
            >
              <History />
            </Button>
          </div>
          {canEdit && (
            <Popover open={editing} onOpenChange={setEditing}>
              <PopoverTrigger asChild>
                <Button
                  variant="ghost"
                  className="vd:mt-3 vd:px-0 vd:text-meta vd:text-mute"
                  onClick={() => setDraft(intro)}
                >
                  <Pencil />
                  Edit introduction
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start">
                <label htmlFor="intro" className="vd:text-ui vd:font-medium">
                  Introduction
                </label>
                <textarea
                  id="intro"
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  rows={5}
                  className="vd:mt-3 vd:w-full vd:resize-y vd:rounded-control vd:bg-deep vd:p-3 vd:font-prose vd:text-ui"
                />
                <div className="vd:mt-3 vd:flex vd:justify-end vd:gap-2">
                  <Button variant="secondary" onClick={() => setEditing(false)}>
                    Cancel
                  </Button>
                  <Button
                    onClick={() => {
                      setIntro(draft);
                      setEditing(false);
                      setSaved(true);
                    }}
                  >
                    Save preview
                  </Button>
                </div>
                <p className="vd:mt-3 vd:text-meta vd:text-mute">
                  Changes last until you reload this preview.
                </p>
              </PopoverContent>
            </Popover>
          )}
          {saved && (
            <output className="vd:block vd:text-meta vd:text-accent">
              Introduction updated in this preview.
            </output>
          )}
        </div>
      </div>
      <AnimatePresence>
        {history && (
          <motion.section
            initial={{ opacity: 0, y: reduced ? 0 : 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reduced ? 0 : 0.2 }}
            className="vd:mb-8 vd:rounded-plate vd:bg-deep vd:p-6"
            aria-label="Update history"
          >
            <div className="vd:flex vd:items-center vd:justify-between">
              <h2 className="vd:text-section vd:font-medium">Update history</h2>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Close history"
                onClick={() => setHistory(false)}
              >
                <X />
              </Button>
            </div>
            {asset.releases.length ? (
              asset.releases.map((release) => (
                <div
                  key={release.version}
                  className="vd:grid vd:gap-2 vd:py-4 vd:md:grid-cols-[11rem_1fr]"
                >
                  <p className="vd:text-ui">
                    Version {release.version}
                    <span className="vd:block vd:text-meta vd:text-mute">
                      {release.date}
                    </span>
                  </p>
                  <div>
                    <p className="vd:text-ui">{release.summary}</p>
                    <ul className="vd:mt-2 vd:space-y-1 vd:text-meta vd:text-mute">
                      {release.changes.map((change) => (
                        <li key={change}>{change}</li>
                      ))}
                    </ul>
                  </div>
                </div>
              ))
            ) : (
              <p className="vd:py-4 vd:text-mute">No published updates yet.</p>
            )}
          </motion.section>
        )}
      </AnimatePresence>
      {asset.blocks.length > 0 && (
        <nav
          aria-label="Page contents"
          className="vd:sticky vd:top-0 vd:z-20 vd:mb-10 vd:flex vd:items-center vd:gap-7 vd:overflow-x-auto vd:bg-field vd:py-3 vd:shadow-[0_8px_15px_-15px_rgb(0_0_0/0.5)]"
        >
          <span className="vd:flex vd:min-h-11 vd:shrink-0 vd:items-center vd:gap-2 vd:text-meta vd:text-mute">
            On this page
            <ChevronDown className="vd:size-3" />
          </span>
          {asset.blocks.map((block) => (
            <a
              key={block.id}
              href={`#block-${block.id}`}
              className="vd:flex vd:min-h-11 vd:shrink-0 vd:items-center vd:text-ui vd:hover:text-accent"
            >
              {block.title}
            </a>
          ))}
        </nav>
      )}
      <div ref={ref} className="vd:grid vd:gap-y-14">
        {rows.map((row) => (
          <div
            key={row[0].block.id}
            className="vd:grid vd:grid-cols-12 vd:gap-y-14"
            style={{ columnGap: BLOCK_GRID_GAP_PX }}
          >
            {row.map(({ block, columns, startColumn }) => (
              <div
                key={block.id}
                data-columns={columns}
                className="vd:min-w-0"
                style={{ gridColumn: `${startColumn} / span ${columns}` }}
              >
                <ReaderBlock block={block} />
              </div>
            ))}
          </div>
        ))}
        {!asset.blocks.length && (
          <p className="vd:py-12 vd:text-mute">
            The creator has not added any page content yet.
          </p>
        )}
      </div>
    </main>
  );
}
