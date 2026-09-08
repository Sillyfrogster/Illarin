"use client";

import { motion, useReducedMotion } from "framer-motion";
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronDown,
  Copy,
  Rss,
} from "lucide-react";
import Image from "next/image";
import { useMemo, useRef, useState } from "react";
import blogDark from "@/assets/art/full/illarin-blog-masthead-dark-v1.webp";
import blogLight from "@/assets/art/full/illarin-blog-masthead-light-v1.webp";
import { Button } from "./components/button";
import { Popover, PopoverContent, PopoverTrigger } from "./components/popover";
import { headings, useReadingPosition } from "./contents";
import { type Body, CATEGORY_LABEL, type Post } from "./data";
import { cn, Inline } from "./ui";

type LinkTo = (surface: "blog" | "article" | "asset", post?: string) => string;

export function StudioBlog({
  posts,
  theme,
  href,
  initialCategory = "all",
  initialApp = "",
}: {
  posts: Post[];
  theme: "light" | "dark";
  href: LinkTo;
  initialCategory?: string;
  initialApp?: string;
}) {
  const [category, setCategory] = useState(initialCategory);
  const [hovered, setHovered] = useState<string | null>(null);
  const reduced = useReducedMotion();
  const selectCategory = (value: string) => {
    setCategory(value);
    const query = new URLSearchParams(window.location.search);
    if (value === "all") query.delete("category");
    else query.set("category", value);
    window.history.replaceState(null, "", `?${query}${window.location.hash}`);
  };
  const app = posts.find((post) => post.app?.slug === initialApp)?.app;
  const filtered = posts.filter(
    (post) =>
      (category === "all" || post.category === category) &&
      (!initialApp || post.app?.slug === initialApp),
  );
  const lead = filtered[0];
  return (
    <main
      id="main-content"
      className="vd:px-5 vd:pb-16 vd:md:px-10 vd:xl:px-14"
    >
      <div className="vd:flex vd:flex-wrap vd:items-end vd:justify-between vd:gap-5 vd:pt-14 vd:pb-8">
        <h1 className="vd:font-display vd:text-display vd:font-medium vd:tracking-[-0.035em]">
          {app ? app.name : "The blog"}
          <span className="vd:text-accent">.</span>
        </h1>
        <p className="vd:max-w-[30ch] vd:text-ui vd:text-mute">
          Updates, experiments and the things
          <br className="vd:hidden vd:sm:block" /> we learn along the way.
        </p>
      </div>
      <div className="vd:grid vd:gap-8 vd:lg:grid-cols-[11rem_minmax(0,1fr)] vd:lg:gap-12">
        <aside className="vd:flex vd:gap-2 vd:overflow-x-auto vd:lg:flex-col vd:lg:items-start vd:lg:gap-1 vd:lg:pt-3">
          {(["all", "announcement", "article", "release"] as const).map(
            (value) => (
              <button
                key={value}
                type="button"
                aria-pressed={category === value}
                onClick={() => selectCategory(value)}
                className={cn(
                  "vd:relative vd:flex vd:min-h-11 vd:shrink-0 vd:items-center vd:gap-3 vd:px-1 vd:text-ui vd:transition-colors",
                  category === value
                    ? "vd:text-accent"
                    : "vd:text-mute vd:hover:text-ink",
                )}
              >
                <motion.span
                  animate={{
                    scale: category === value ? 1 : 0.5,
                    opacity: category === value ? 1 : 0.35,
                  }}
                  transition={{ duration: reduced ? 0 : 0.2 }}
                  className="vd:size-1.5 vd:rounded-full vd:bg-current"
                />
                {value === "all" ? "Everything" : CATEGORY_LABEL[value]}
              </button>
            ),
          )}
          <a
            href="/blog/feed.xml"
            className="vd:mt-0 vd:flex vd:min-h-11 vd:shrink-0 vd:items-center vd:gap-3 vd:px-1 vd:text-meta vd:text-mute vd:lg:mt-12"
          >
            <Rss className="vd:size-4" />
            Follow the feed
          </a>
        </aside>
        <div className="vd:min-w-0">
          {initialApp && (
            <a
              href={href("blog")}
              className="vd:mb-5 vd:inline-flex vd:min-h-11 vd:items-center vd:gap-2 vd:text-meta vd:text-mute"
            >
              <ArrowLeft className="vd:size-4" />
              All publication apps
            </a>
          )}
          {lead ? (
            <article className="vd:grid vd:gap-7 vd:xl:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)] vd:xl:items-center">
              {lead.image && (
                <a
                  href={href("article", lead.id)}
                  className="vd:group vd:relative vd:block vd:overflow-hidden vd:rounded-plate vd:bg-deep"
                >
                  <Image
                    src={lead.image}
                    alt="Illustration accompanying the latest article"
                    priority
                    sizes="(max-width: 1024px) 90vw, 50vw"
                    className="vd:aspect-[5/4] vd:h-auto vd:w-full vd:object-contain vd:transition-transform vd:duration-500 vd:group-hover:scale-[1.02] vd:motion-reduce:transform-none vd:sm:max-xl:max-h-[360px]"
                  />
                  <span className="vd:absolute vd:right-4 vd:bottom-4 vd:flex vd:size-12 vd:items-center vd:justify-center vd:rounded-full vd:bg-field vd:text-ink">
                    <ArrowRight className="vd:size-5 vd:-rotate-45 vd:transition-transform vd:group-hover:rotate-0" />
                  </span>
                </a>
              )}
              <div
                className={cn(
                  "vd:min-w-0 vd:py-3",
                  !lead.image && "vd:xl:col-span-2 vd:xl:max-w-[52rem]",
                )}
              >
                <h2 className="vd:max-w-[24ch] vd:font-display vd:text-[clamp(1.8rem,3vw,3rem)] vd:leading-[1.13] vd:font-medium vd:tracking-[-0.025em]">
                  <a
                    href={href("article", lead.id)}
                    className="vd:hover:text-accent"
                  >
                    {lead.title}
                  </a>
                </h2>
                <p className="vd:mt-4 vd:text-meta vd:text-mute">
                  {CATEGORY_LABEL[lead.category]}{" "}
                  <span aria-hidden="true">·</span> {lead.date}
                </p>
                <p className="vd:mt-6 vd:max-w-[42ch] vd:font-prose vd:text-ui vd:leading-7 vd:text-mute">
                  {lead.dek}
                </p>
                <a
                  href={href("article", lead.id)}
                  className="vd:mt-5 vd:inline-flex vd:min-h-11 vd:items-center vd:gap-3 vd:text-ui vd:font-medium vd:text-accent"
                >
                  Read the story
                  <ArrowRight className="vd:size-4" />
                </a>
              </div>
            </article>
          ) : (
            <div className="vd:py-16">
              <h2 className="vd:text-title">No posts here yet.</h2>
              <p className="vd:mt-3 vd:text-mute">
                Try another category to keep reading.
              </p>
              <Button
                variant="secondary"
                className="vd:mt-5"
                onClick={() => selectCategory("all")}
              >
                Show everything
              </Button>
            </div>
          )}
          {filtered.length > 1 && (
            <div className="vd:mt-14">
              <h2 className="vd:mb-4 vd:text-section vd:font-medium">
                Earlier
              </h2>
              <div className="vd:relative">
                {filtered.slice(1).map((post) => (
                  <article
                    key={post.id}
                    className="vd:relative"
                    onMouseEnter={() => setHovered(post.id)}
                    onMouseLeave={() => setHovered(null)}
                  >
                    {hovered === post.id && (
                      <motion.div
                        layoutId="story-highlight"
                        className="vd:absolute vd:-inset-x-4 vd:inset-y-1 vd:rounded-plate vd:bg-deep"
                        transition={
                          reduced
                            ? { duration: 0 }
                            : { type: "spring", stiffness: 260, damping: 30 }
                        }
                      />
                    )}
                    <a
                      href={href("article", post.id)}
                      onFocus={() => setHovered(post.id)}
                      onBlur={() => setHovered(null)}
                      className="vd:relative vd:grid vd:items-start vd:gap-4 vd:py-6 vd:sm:grid-cols-[7.5rem_minmax(0,1fr)_1.5rem] vd:sm:gap-7"
                    >
                      <span className="vd:text-meta vd:text-mute">
                        {post.date}
                        <span className="vd:mt-1 vd:block vd:text-accent">
                          {CATEGORY_LABEL[post.category]}
                        </span>
                      </span>
                      <div className="vd:min-w-0">
                        <h3 className="vd:max-w-[44ch] vd:font-display vd:text-section vd:font-medium vd:leading-snug">
                          {post.title}
                        </h3>
                        <p className="vd:mt-2 vd:max-w-[64ch] vd:font-prose vd:text-ui vd:leading-6 vd:text-mute">
                          {post.dek}
                        </p>
                      </div>
                      <ArrowRight className="vd:hidden vd:size-5 vd:sm:block" />
                    </a>
                  </article>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
      <div className="vd:relative vd:mt-16 vd:overflow-hidden vd:rounded-plate vd:bg-deep">
        <Image
          src={theme === "dark" ? blogDark : blogLight}
          alt=""
          sizes="100vw"
          className="vd:h-36 vd:w-full vd:object-cover vd:object-right"
        />
        <a
          href="/browse"
          className="vd:absolute vd:inset-y-0 vd:left-6 vd:flex vd:items-center vd:gap-4 vd:text-section vd:font-medium"
        >
          Back to the collection
          <ArrowRight className="vd:size-5" />
        </a>
      </div>
    </main>
  );
}

function ArticleBody({ blocks }: { blocks: Body[] }) {
  let heading = 0;
  return (
    <div className="vd:font-prose vd:text-[1.0625rem] vd:leading-[1.85] vd:[&>p+p]:mt-5">
      {blocks.map((block, index) => {
        const key = `${block.kind}-${index}`;
        if (block.kind === "h") {
          heading++;
          return (
            <h2
              key={key}
              id={`section-${heading}`}
              tabIndex={-1}
              className="vd:mt-12 vd:mb-5 vd:scroll-mt-8 vd:font-display vd:text-title vd:font-medium vd:leading-tight"
            >
              <Inline text={block.text} />
            </h2>
          );
        }
        if (block.kind === "p")
          return (
            <p key={key}>
              <Inline text={block.text} />
            </p>
          );
        if (block.kind === "figure")
          return (
            <figure key={key} className="vd:my-9">
              <Image
                src={block.image}
                alt={block.alt}
                sizes="(max-width: 768px) 90vw, 720px"
                className="vd:h-auto vd:w-full vd:rounded-plate"
              />
              <figcaption className="vd:mt-3 vd:text-meta vd:leading-6 vd:text-mute">
                {block.caption}
              </figcaption>
            </figure>
          );
        if (block.kind === "quote")
          return (
            <blockquote
              key={key}
              className="vd:my-10 vd:font-display vd:text-title vd:font-normal vd:leading-snug vd:text-accent"
            >
              “{block.text}”
            </blockquote>
          );
        if (block.kind === "list")
          return (
            <ul key={key} className="vd:my-5 vd:list-disc vd:space-y-3 vd:pl-5">
              {block.items.map((item) => (
                <li key={item}>
                  <Inline text={item} />
                </li>
              ))}
            </ul>
          );
        if (block.kind === "code")
          return (
            <pre
              key={key}
              className="vd:my-8 vd:overflow-x-auto vd:rounded-control vd:bg-deep vd:p-5 vd:text-meta vd:leading-7"
            >
              <code>{block.lines.join("\n")}</code>
            </pre>
          );
        if (block.kind === "table")
          return (
            <div key={key} className="vd:my-8 vd:overflow-x-auto">
              <table className="vd:w-full vd:min-w-[520px] vd:border-collapse vd:text-left vd:text-meta">
                <thead>
                  <tr>
                    {block.head.map((cell) => (
                      <th
                        key={cell}
                        scope="col"
                        className="vd:bg-deep vd:px-3 vd:py-3 vd:font-medium"
                      >
                        {cell}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {block.rows.map((row) => (
                    <tr key={row[0]} className="vd:even:bg-deep/50">
                      {row.map((cell, i) => (
                        <td key={`${i}-${cell}`} className="vd:px-3 vd:py-3">
                          {cell}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        return null;
      })}
    </div>
  );
}

export function StudioArticle({
  post,
  posts,
  href,
}: {
  post: Post;
  posts: Post[];
  href: LinkTo;
}) {
  const list = useMemo(() => headings(post.body), [post.body]);
  const active = useReadingPosition(list);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  const [contentsOpen, setContentsOpen] = useState(false);
  const contentsTarget = useRef<HTMLElement | null>(null);
  const related = posts
    .filter((candidate) => candidate.id !== post.id)
    .sort((a, b) => {
      const rank = (candidate: Post) =>
        (post.app && candidate.app?.slug === post.app.slug ? 2 : 0) +
        (candidate.category === post.category ? 1 : 0);
      return rank(b) - rank(a);
    })
    .slice(0, 3);
  const contents = (
    <nav aria-label="Article contents" className="vd:space-y-1">
      {list.map((item) => (
        <a
          key={item.id}
          href={`#${item.id}`}
          onClick={() => {
            if (contentsOpen) {
              contentsTarget.current = document.getElementById(item.id);
              setContentsOpen(false);
            }
          }}
          className={cn(
            "vd:flex vd:min-h-11 vd:items-center vd:py-2 vd:text-ui",
            active === item.id ? "vd:text-accent" : "vd:text-mute",
          )}
          aria-current={active === item.id ? "location" : undefined}
        >
          {item.text}
        </a>
      ))}
    </nav>
  );
  return (
    <main
      id="main-content"
      className="vd:px-5 vd:pb-16 vd:md:px-10 vd:xl:px-14"
    >
      <a
        href={href("blog")}
        className="vd:mt-7 vd:inline-flex vd:min-h-11 vd:items-center vd:gap-2 vd:text-meta vd:text-mute"
      >
        <ArrowLeft className="vd:size-4" />
        All posts
      </a>
      <header className="vd:mx-auto vd:max-w-[64rem] vd:pt-9 vd:pb-10">
        <h1 className="vd:max-w-[25ch] vd:font-display vd:text-hero vd:font-medium vd:tracking-[-0.03em] vd:break-words">
          {post.title}
        </h1>
        <p className="vd:mt-7 vd:max-w-[54ch] vd:font-prose vd:text-lede vd:text-mute">
          {post.dek}
        </p>
        <div className="vd:mt-8 vd:flex vd:flex-wrap vd:items-center vd:gap-x-5 vd:gap-y-2 vd:text-meta">
          <span className="vd:text-accent">
            {CATEGORY_LABEL[post.category]}
          </span>
          <span className="vd:inline-flex vd:flex-wrap vd:items-center vd:gap-1">
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant="link"
                  className="vd:h-auto vd:min-h-11 vd:px-0 vd:text-meta vd:font-normal vd:text-ink"
                  aria-label={`Preview ${post.author}'s profile`}
                >
                  {post.author}
                </Button>
              </PopoverTrigger>
              <PopoverContent aria-label="Synthetic author profile">
                <p className="vd:text-section vd:font-medium">{post.author}</p>
                <p className="vd:mt-1 vd:text-meta vd:text-mute">
                  {post.handle}
                </p>
                {post.role && <p className="vd:mt-3 vd:text-ui">{post.role}</p>}
                <p className="vd:mt-3 vd:text-meta vd:text-mute">
                  Synthetic profile for this preview.
                </p>
              </PopoverContent>
            </Popover>
            {post.role && <span>· {post.role}</span>}
          </span>
          <span className="vd:text-mute">{post.date}</span>
          {post.updated && (
            <span className="vd:text-mute">Updated {post.updated}</span>
          )}
          {post.app && (
            <a
              href={`${href("blog")}&app=${encodeURIComponent(post.app.slug)}`}
              className="vd:text-mute"
            >
              {post.app.name}
            </a>
          )}
          {post.category === "release" && post.version && (
            <span>Version {post.version}</span>
          )}
        </div>
      </header>
      {post.image && (
        <figure className="vd:mx-auto vd:mb-12 vd:max-w-[64rem] vd:overflow-hidden vd:rounded-plate vd:bg-deep">
          <Image
            src={post.image}
            alt="Article illustration"
            priority
            sizes="(max-width: 1024px) 90vw, 1024px"
            className="vd:max-h-[460px] vd:w-full vd:object-contain"
          />
        </figure>
      )}
      <div className="vd:mx-auto vd:grid vd:max-w-[64rem] vd:items-start vd:gap-10 vd:lg:grid-cols-[13rem_minmax(0,1fr)] vd:lg:gap-12">
        <aside className="vd:lg:sticky vd:lg:top-8">
          {list.length >= 3 && (
            <>
              <div className="vd:hidden vd:lg:block">
                <h2 className="vd:mb-3 vd:text-meta vd:font-medium">
                  In this article
                </h2>
                {contents}
              </div>
              <div className="vd:lg:hidden">
                <Popover open={contentsOpen} onOpenChange={setContentsOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="secondary"
                      className="vd:w-full vd:justify-between"
                    >
                      In this article
                      <ChevronDown />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    onCloseAutoFocus={(event) => {
                      if (contentsTarget.current) {
                        event.preventDefault();
                        contentsTarget.current.focus({ preventScroll: true });
                        contentsTarget.current = null;
                      }
                    }}
                  >
                    {contents}
                  </PopoverContent>
                </Popover>
              </div>
            </>
          )}
          <Button
            variant="ghost"
            className="vd:mt-4 vd:px-0 vd:text-meta vd:text-mute"
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(window.location.href);
                setCopied(true);
                setError("");
              } catch {
                setError("Could not copy. Select the address in your browser.");
              }
            }}
          >
            {copied ? <Check /> : <Copy />}
            {copied ? "Link copied" : "Copy article link"}
          </Button>
          {error && (
            <p role="alert" className="vd:text-meta vd:text-stop">
              {error}
            </p>
          )}
        </aside>
        <div className="vd:min-w-0 vd:max-w-[70ch]">
          <ArticleBody blocks={post.body ?? [{ kind: "p", text: post.dek }]} />
          <div className="vd:mt-14 vd:flex vd:flex-wrap vd:items-center vd:justify-between vd:gap-4 vd:rounded-control vd:bg-deep vd:p-5">
            <p className="vd:text-ui">Keep up with Illarin.</p>
            <a
              href="/blog/feed.xml"
              className="vd:flex vd:min-h-11 vd:items-center vd:gap-2 vd:text-ui"
            >
              <Rss className="vd:size-4" />
              Follow the feed
            </a>
          </div>
          <h2 className="vd:mt-12 vd:mb-4 vd:text-section vd:font-medium">
            Keep reading
          </h2>
          {related.map((p) => (
            <a
              key={p.id}
              href={href("article", p.id)}
              className="vd:flex vd:min-h-14 vd:items-center vd:justify-between vd:gap-5 vd:py-3 vd:text-ui vd:hover:text-accent"
            >
              {p.title}
              <ArrowRight className="vd:size-4 vd:shrink-0" />
            </a>
          ))}
          <a
            href="#main-content"
            className="vd:mt-8 vd:inline-flex vd:min-h-11 vd:items-center vd:gap-2 vd:text-meta vd:text-mute"
          >
            Back to top
            <ArrowDown className="vd:size-4 vd:rotate-180" />
          </a>
        </div>
      </div>
    </main>
  );
}
