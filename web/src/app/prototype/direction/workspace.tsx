"use client";

import { useEffect, useState } from "react";
import { AmbientArticle } from "./ambient/article";
import { AmbientAsset } from "./ambient/asset";
import { AmbientBlog } from "./ambient/blog";
import { ASSETS, KIND_LABEL, type Kind } from "./assets";
import { DARK_TINT, RICH_POSTS, SPARSE_POSTS } from "./data";
import { LedgerArticle } from "./ledger/article";
import { LedgerAsset } from "./ledger/asset";
import { LedgerBlog } from "./ledger/blog";
import { cn, type Direction, tintStyle } from "./ui";
import { VitrineArticle } from "./vitrine/article";
import { VitrineAsset } from "./vitrine/asset";
import { VitrineBlog } from "./vitrine/blog";

type Surface = "blog" | "article" | "asset";
type Content = "rich" | "sparse";
type AssetPick = Kind | "sparse";
type Theme = "light" | "dark";

const DIRECTIONS: { id: Direction; name: string; line: string }[] = [
  { id: "vitrine", name: "Vitrine", line: "Light is the material" },
  { id: "ambient", name: "Ambient", line: "The work colours the room" },
  { id: "ledger", name: "Ledger", line: "Typeset and indexed" },
];

const SURFACES: { id: Surface; name: string }[] = [
  { id: "blog", name: "Blog" },
  { id: "article", name: "Article" },
  { id: "asset", name: "Asset" },
];

const ASSET_PICKS: { id: AssetPick; name: string }[] = [
  { id: "character", name: KIND_LABEL.character },
  { id: "lorebook", name: KIND_LABEL.lorebook },
  { id: "preset", name: KIND_LABEL.preset },
  { id: "theme", name: KIND_LABEL.theme },
  { id: "pack", name: KIND_LABEL.pack },
  { id: "sparse", name: "Sparse" },
];

function readSetting<T extends string>(
  key: string,
  allowed: readonly T[],
  fallback: T,
): T {
  if (typeof window === "undefined") return fallback;
  const value = new URLSearchParams(window.location.search).get(key);
  return allowed.includes(value as T) ? (value as T) : fallback;
}

function Segment<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: T;
  options: { id: T; name: string }[];
  onChange: (next: T) => void;
}) {
  return (
    <div className="vd:flex vd:shrink-0 vd:items-center vd:gap-2">
      <span className="vd:hidden vd:text-[0.6875rem] vd:font-bold vd:uppercase vd:tracking-[0.14em] vd:text-white/40 vd:xl:block">
        {label}
      </span>
      <fieldset
        aria-label={label}
        className="vd:flex vd:items-center vd:gap-0.5 vd:rounded-full vd:bg-white/10 vd:p-0.5"
      >
        {options.map((option) => (
          <button
            key={option.id}
            type="button"
            aria-pressed={option.id === value}
            onClick={() => onChange(option.id)}
            className={cn(
              "vd:min-h-9 vd:rounded-full vd:px-3.5 vd:text-[0.8125rem] vd:font-semibold vd:whitespace-nowrap vd:transition",
              option.id === value
                ? "vd:bg-white vd:text-neutral-900"
                : "vd:text-white/70 vd:hover:text-white",
            )}
          >
            {option.name}
          </button>
        ))}
      </fieldset>
    </div>
  );
}

export function DirectionPrototype() {
  const [direction, setDirection] = useState<Direction>("vitrine");
  const [surface, setSurface] = useState<Surface>("blog");
  const [content, setContent] = useState<Content>("rich");
  const [pick, setPick] = useState<AssetPick>("character");
  const [theme, setTheme] = useState<Theme>("light");

  const [ready, setReady] = useState(false);
  const [showControls, setShowControls] = useState(true);

  useEffect(() => {
    setDirection(
      readSetting("d", ["vitrine", "ambient", "ledger"] as const, "vitrine"),
    );
    setSurface(readSetting("s", ["blog", "article", "asset"] as const, "blog"));
    setContent(readSetting("c", ["rich", "sparse"] as const, "rich"));
    setPick(
      readSetting(
        "k",
        ["character", "lorebook", "preset", "theme", "pack", "sparse"] as const,
        "character",
      ),
    );
    setTheme(readSetting("t", ["light", "dark"] as const, "light"));
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    const query = new URLSearchParams({
      d: direction,
      s: surface,
      c: content,
      k: pick,
      t: theme,
    });
    window.history.replaceState(null, "", `?${query}`);
  }, [ready, direction, surface, content, pick, theme]);

  const posts = content === "rich" ? RICH_POSTS : SPARSE_POSTS;
  const asset = ASSETS[pick];
  const article = posts[0];

  const pageTint =
    surface === "asset"
      ? asset.tint
      : surface === "article"
        ? article?.tint
        : undefined;

  return (
    <div
      data-direction={direction}
      data-theme={theme}
      style={tintStyle(
        pageTint ?? (theme === "dark" ? DARK_TINT : undefined),
        theme === "dark",
      )}
      className="vd:min-h-dvh"
    >
      {surface === "blog" && direction === "vitrine" && (
        <VitrineBlog posts={posts} theme={theme} />
      )}
      {surface === "blog" && direction === "ambient" && (
        <AmbientBlog posts={posts} theme={theme} />
      )}
      {surface === "blog" && direction === "ledger" && (
        <LedgerBlog posts={posts} />
      )}

      {surface === "article" && direction === "vitrine" && (
        <VitrineArticle post={article} />
      )}
      {surface === "article" && direction === "ambient" && (
        <AmbientArticle post={article} theme={theme} />
      )}
      {surface === "article" && direction === "ledger" && (
        <LedgerArticle post={article} />
      )}

      {surface === "asset" && direction === "vitrine" && (
        <VitrineAsset asset={asset} theme={theme} />
      )}
      {surface === "asset" && direction === "ambient" && (
        <AmbientAsset asset={asset} theme={theme} />
      )}
      {surface === "asset" && direction === "ledger" && (
        <LedgerAsset asset={asset} theme={theme} />
      )}

      {/* The study's own controls stay clear of the centre, where a page may dock its own */}
      <div className="vd:fixed vd:right-3 vd:bottom-3 vd:z-50 vd:flex vd:max-w-[calc(100vw-1.5rem)] vd:flex-col vd:items-end vd:gap-2">
        {showControls && (
          <div className="vd:flex vd:max-w-[26rem] vd:flex-wrap vd:justify-end vd:gap-2 vd:rounded-3xl vd:bg-neutral-900 vd:p-2 vd:shadow-[0_18px_40px_-18px_rgb(0_0_0/0.7)]">
            <Segment
              label="Direction"
              value={direction}
              options={DIRECTIONS}
              onChange={setDirection}
            />
            <Segment
              label="Surface"
              value={surface}
              options={SURFACES}
              onChange={setSurface}
            />
            {surface === "asset" ? (
              <Segment
                label="Asset"
                value={pick}
                options={ASSET_PICKS}
                onChange={setPick}
              />
            ) : (
              <Segment
                label="Content"
                value={content}
                options={[
                  { id: "rich" as Content, name: "Rich" },
                  { id: "sparse" as Content, name: "Sparse" },
                ]}
                onChange={setContent}
              />
            )}
            <Segment
              label="Theme"
              value={theme}
              options={[
                { id: "light" as Theme, name: "Light" },
                { id: "dark" as Theme, name: "Dark" },
              ]}
              onChange={setTheme}
            />
          </div>
        )}
        <button
          type="button"
          aria-expanded={showControls}
          onClick={() => setShowControls(!showControls)}
          className="vd:inline-flex vd:min-h-9 vd:shrink-0 vd:items-center vd:rounded-full vd:bg-neutral-900 vd:px-4 vd:text-[0.6875rem] vd:font-bold vd:tracking-[0.14em] vd:text-white/70 vd:uppercase vd:shadow-[0_18px_40px_-18px_rgb(0_0_0/0.7)] vd:hover:text-white"
        >
          {showControls ? "Hide study controls" : "Direction study"}
        </button>
      </div>
    </div>
  );
}
