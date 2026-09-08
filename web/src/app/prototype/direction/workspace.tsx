"use client";

import { MotionConfig } from "framer-motion";
import { FlaskConical, Settings2 } from "lucide-react";
import { useEffect, useState } from "react";
import { ASSETS } from "./assets";
import { Button } from "./components/button";
import { NotchNav } from "./components/notch-nav";
import { Popover, PopoverContent, PopoverTrigger } from "./components/popover";
import { RICH_POSTS, SPARSE_POSTS } from "./data";
import { StudioArticle, StudioBlog } from "./studio-publication";
import { StudioReader } from "./studio-reader";

type Surface = "blog" | "article" | "asset";
type Theme = "light" | "dark";

export function DirectionPrototype() {
  const [surface, setSurface] = useState<Surface>("asset");
  const [theme, setTheme] = useState<Theme>("dark");
  const [pick, setPick] = useState("character");
  const [sparse, setSparse] = useState(false);
  const [postId, setPostId] = useState("");
  const [category, setCategory] = useState("all");
  const [app, setApp] = useState("");
  const [readerState, setReaderState] = useState("owner");
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    const s = q.get("s");
    if (s === "blog" || s === "article" || s === "asset") setSurface(s);
    setTheme(q.get("t") === "light" ? "light" : "dark");
    const kind = q.get("k");
    if (kind && Object.hasOwn(ASSETS, kind)) setPick(kind);
    setSparse(q.get("c") === "sparse");
    setPostId(q.get("p") ?? "");
    setCategory(q.get("category") ?? "all");
    setApp(q.get("app") ?? "");
    setReaderState(q.get("state") ?? "owner");
    setReady(true);
  }, []);
  const href = (next: Surface, post?: string) => {
    const query = new URLSearchParams({
      s: next,
      t: theme,
      k: pick,
      c: sparse ? "sparse" : "rich",
    });
    if (post) query.set("p", post);
    return `/prototype/direction?${query}`;
  };
  useEffect(() => {
    if (!ready) return;
    const query = new URLSearchParams(window.location.search);
    query.delete("d");
    query.set("s", surface);
    query.set("t", theme);
    query.set("k", pick);
    query.set("c", sparse ? "sparse" : "rich");
    query.set("state", readerState);
    window.history.replaceState(null, "", `?${query}${window.location.hash}`);
  }, [ready, surface, theme, pick, sparse, readerState]);
  const posts = sparse ? SPARSE_POSTS : RICH_POSTS;
  const post = posts.find((item) => item.id === postId) ?? posts[0];
  const asset = ASSETS[sparse ? "sparse" : pick];
  return (
    <MotionConfig reducedMotion="user">
      <div
        data-direction="studio"
        data-theme={theme}
        aria-busy={!ready}
        className="vd:min-h-dvh"
      >
        <a
          href="#main-content"
          className="vd:sr-only vd:focus:not-sr-only vd:focus:absolute vd:focus:z-50 vd:focus:bg-plane vd:focus:p-4"
        >
          Skip to content
        </a>
        <NotchNav
          theme={theme}
          onTheme={() => setTheme(theme === "dark" ? "light" : "dark")}
          blog={surface !== "asset"}
          href={href}
        />
        <div className="vd:mx-5 vd:mt-5 vd:flex vd:items-center vd:justify-between vd:gap-4 vd:text-meta vd:text-mute vd:md:mx-10 vd:xl:mx-14">
          <span className="vd:flex vd:items-center vd:gap-2">
            <FlaskConical className="vd:size-3.5" />
            Interactive study · synthetic content
          </span>
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="ghost" className="vd:gap-2 vd:px-2 vd:text-meta">
                <Settings2 />
                Preview settings
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end">
              <h2 className="vd:text-section vd:font-medium">
                Explore the direction
              </h2>
              <label
                htmlFor="surface"
                className="vd:mt-5 vd:block vd:text-meta"
              >
                Page
              </label>
              <select
                id="surface"
                value={surface}
                onChange={(e) => setSurface(e.target.value as Surface)}
                className="vd:mt-2 vd:h-11 vd:w-full vd:rounded-control vd:bg-deep vd:px-3 vd:text-ui"
              >
                <option value="asset">Asset reader</option>
                <option value="blog">Blog discovery</option>
                <option value="article">Article reading</option>
              </select>
              {surface === "asset" && (
                <>
                  <label
                    htmlFor="kind"
                    className="vd:mt-4 vd:block vd:text-meta"
                  >
                    Asset kind
                  </label>
                  <select
                    id="kind"
                    value={pick}
                    onChange={(e) => setPick(e.target.value)}
                    className="vd:mt-2 vd:h-11 vd:w-full vd:rounded-control vd:bg-deep vd:px-3 vd:text-ui"
                  >
                    {Object.keys(ASSETS)
                      .filter((k) => k !== "sparse")
                      .map((k) => (
                        <option key={k} value={k}>
                          {k[0].toUpperCase() + k.slice(1)}
                        </option>
                      ))}
                  </select>
                  <label
                    htmlFor="reader-state"
                    className="vd:mt-4 vd:block vd:text-meta"
                  >
                    Reader scenario
                  </label>
                  <select
                    id="reader-state"
                    value={readerState}
                    onChange={(e) => setReaderState(e.target.value)}
                    className="vd:mt-2 vd:h-11 vd:w-full vd:rounded-control vd:bg-deep vd:px-3 vd:text-ui"
                  >
                    <option value="owner">Creator can edit</option>
                    <option value="visitor">Visitor can read</option>
                    <option value="failure">Download fails once</option>
                  </select>
                </>
              )}
              <label className="vd:mt-4 vd:flex vd:min-h-11 vd:items-center vd:gap-3 vd:text-ui">
                <input
                  type="checkbox"
                  checked={sparse}
                  onChange={(e) => setSparse(e.target.checked)}
                  className="vd:size-4 vd:accent-accent"
                />
                Sparse content, without a cover
              </label>
              <Button asChild variant="secondary" className="vd:mt-4 vd:w-full">
                <a href="/prototype/editor">Open the editing workspace</a>
              </Button>
              <p className="vd:mt-3 vd:text-meta vd:text-mute">
                A separate working example. Production pages are unchanged.
              </p>
            </PopoverContent>
          </Popover>
        </div>
        {surface === "asset" && (
          <StudioReader
            key={`${asset.id}-${readerState}`}
            asset={asset}
            canEdit={readerState !== "visitor"}
            failOnce={readerState === "failure"}
          />
        )}
        {surface === "blog" && (
          <StudioBlog
            key={`${sparse}-${category}-${app}`}
            posts={posts}
            theme={theme}
            href={href}
            initialCategory={category}
            initialApp={app}
          />
        )}
        {surface === "article" && (
          <StudioArticle key={post.id} post={post} posts={posts} href={href} />
        )}
        <footer className="vd:mx-5 vd:flex vd:flex-wrap vd:items-center vd:justify-between vd:gap-5 vd:py-8 vd:text-meta vd:text-mute vd:md:mx-10 vd:xl:mx-14">
          <a
            href="/"
            className="vd:flex vd:min-h-11 vd:items-center vd:font-display vd:text-section vd:font-medium vd:text-ink"
          >
            illarin.
          </a>
          <span>A home for the things you make.</span>
          <div className="vd:flex vd:gap-5">
            <a
              href="/legal/privacy"
              className="vd:flex vd:min-h-11 vd:items-center"
            >
              Privacy
            </a>
            <a
              href="/legal/terms"
              className="vd:flex vd:min-h-11 vd:items-center"
            >
              Terms
            </a>
          </div>
        </footer>
      </div>
    </MotionConfig>
  );
}
