"use client";

import type { CSSProperties } from "react";
import { Body } from "../body";
import { headings, useReadingPosition } from "../contents";
import { CATEGORY_LABEL, CATEGORY_PIGMENT, type Post } from "../data";
import { SiteFoot, SiteHead } from "../shell";
import { Action, cn, Empty, Label, Rule } from "../ui";

function Contents({ post }: { post: Post }) {
  const list = headings(post.body);
  const active = useReadingPosition(list);
  if (list.length < 2) return null;
  return (
    <nav aria-label="Contents" className="vd:lg:sticky vd:lg:top-24">
      <Label>Contents</Label>
      <ol className="vd:mt-4">
        {list.map((entry) => (
          <li key={entry.id}>
            <a
              href={`#${entry.id}`}
              aria-current={entry.id === active ? "true" : undefined}
              className={cn(
                "vd:flex vd:min-h-11 vd:items-baseline vd:gap-3 vd:py-1 vd:text-meta vd:leading-6 vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
                entry.id === active
                  ? "vd:text-ink"
                  : "vd:text-mute vd:hover:text-ink",
              )}
            >
              <span
                className="v-tabular vd:shrink-0"
                style={{
                  color:
                    entry.id === active ? "var(--v-accent)" : "var(--v-faint)",
                }}
              >
                {String(entry.number).padStart(2, "0")}
              </span>
              <span className="vd:min-w-0">{entry.text}</span>
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}

export function LedgerArticle({ post }: { post?: Post }) {
  if (!post) {
    return (
      <div id="top">
        <SiteHead direction="ledger" section="Publication ledger" />
        <div className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
          <Empty title="No entry" detail="Nothing recorded here yet." />
        </div>
      </div>
    );
  }

  const colophon: [string, string][] = [
    ["Category", CATEGORY_LABEL[post.category]],
    ["Author", `${post.author} ${post.handle}`],
    ["Published", post.date],
    ["Last edited", post.updated ?? "Not edited"],
    ["Reading time", `${post.minutes} minutes`],
    ["Sections", String(headings(post.body).length || 1)],
  ];

  return (
    <div
      id="top"
      style={{ "--v-accent": CATEGORY_PIGMENT[post.category] } as CSSProperties}
    >
      <SiteHead direction="ledger" section="Publication ledger" />

      <article className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:pt-14 vd:md:px-10">
        <header>
          <div className="vd:flex vd:items-baseline vd:gap-4">
            <Label style={{ color: "var(--v-accent)" }}>
              {CATEGORY_LABEL[post.category]}
            </Label>
            <span className="v-tabular vd:text-meta vd:text-faint">№ 008</span>
          </div>
          <h1 className="vd:mt-5 vd:max-w-[22ch] vd:font-display vd:text-display vd:font-medium">
            {post.title}
          </h1>
          <p className="v-lede vd:mt-5 vd:font-prose vd:text-lede vd:text-mute">
            {post.dek}
          </p>
          <div className="vd:mt-7 vd:flex vd:flex-wrap vd:items-center vd:justify-between vd:gap-5">
            <p className="vd:flex vd:flex-wrap vd:gap-x-5 vd:text-meta vd:text-mute">
              <span className="vd:font-semibold vd:text-ink">
                {post.author}
              </span>
              <span className="v-tabular">{post.date}</span>
              {post.updated && (
                <span className="v-tabular">updated {post.updated}</span>
              )}
              <span className="v-tabular">{post.minutes} min</span>
            </p>
            <Action variant="secondary">Copy link</Action>
          </div>
          <div className="vd:mt-8">
            <Rule />
          </div>
        </header>

        <div className="vd:mt-group vd:grid vd:gap-x-12 vd:gap-y-10 vd:lg:grid-cols-12">
          <div className="vd:lg:col-span-3">
            <Contents post={post} />
          </div>
          <div
            className="v-article vd:lg:col-span-8 vd:lg:col-start-5"
            style={
              {
                "--col-margin": "0rem",
                "--col-text": "40rem",
              } as CSSProperties
            }
          >
            {post.body ? (
              <Body blocks={post.body} direction="ledger" />
            ) : (
              <Empty
                title="This one is still being written"
                detail="The summary above is all there is for now."
              />
            )}
          </div>
        </div>

        <section className="vd:mt-chapter">
          <Rule />
          <h2 className="vd:mt-6 vd:font-display vd:text-section vd:font-medium">
            Colophon
          </h2>
          <dl className="v-tabular vd:mt-5 vd:grid vd:gap-x-10 vd:sm:grid-cols-2 vd:lg:grid-cols-3">
            {colophon.map(([term, value]) => (
              <div
                key={term}
                className="vd:flex vd:items-baseline vd:gap-3 vd:py-2.5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
              >
                <dt className="vd:shrink-0 vd:text-meta vd:text-mute">
                  {term}
                </dt>
                <span
                  aria-hidden="true"
                  className="v-leader vd:h-px vd:min-w-4 vd:flex-1 vd:self-end vd:mb-[0.4em]"
                />
                <dd className="vd:text-meta vd:font-semibold">{value}</dd>
              </div>
            ))}
          </dl>
        </section>
      </article>

      <SiteFoot />
    </div>
  );
}
