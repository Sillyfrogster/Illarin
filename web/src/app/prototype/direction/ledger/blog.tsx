"use client";

import Image from "next/image";
import type { CSSProperties } from "react";
import {
  CATEGORY_LABEL,
  CATEGORY_PIGMENT,
  type Category,
  type Post,
} from "../data";
import { Counter } from "../motion";
import { SiteFoot, SiteHead } from "../shell";
import { cn, Empty, Label, Rule } from "../ui";

const ORDER: Category[] = ["announcement", "article", "release"];

function Entry({
  post,
  index,
  withMedia,
}: {
  post: Post;
  index: number;
  withMedia: boolean;
}) {
  return (
    <li
      style={{ "--v-accent": CATEGORY_PIGMENT[post.category] } as CSSProperties}
    >
      <a
        href="#top"
        className="v-row vd:group vd:grid vd:grid-cols-[2.5rem_1fr] vd:items-start vd:gap-x-4 vd:gap-y-2 vd:py-6 vd:sm:grid-cols-[3rem_1fr_auto] vd:sm:gap-x-6"
      >
        <span
          className="v-tabular vd:pt-1 vd:text-meta vd:font-semibold"
          style={{ color: "var(--v-accent)" }}
        >
          {String(index).padStart(3, "0")}
        </span>
        <div className="vd:min-w-0">
          <div className="vd:flex vd:items-baseline vd:gap-3">
            <h3 className="vd:font-display vd:text-[clamp(1.375rem,1.9vw,1.75rem)] vd:leading-[1.18] vd:font-medium vd:wrap-pretty">
              {post.title}
            </h3>
            <span
              aria-hidden="true"
              className="vd:relative vd:hidden vd:h-px vd:min-w-6 vd:flex-1 vd:self-end vd:mb-[0.45em] vd:sm:block"
            >
              <span className="v-leader vd:absolute vd:inset-0" />
              <span className="v-leader v-leader-fill vd:absolute vd:inset-0" />
            </span>
            <span className="v-tabular vd:hidden vd:shrink-0 vd:text-meta vd:text-mute vd:sm:block">
              {post.date}
            </span>
          </div>
          <p className="vd:mt-2 vd:max-w-[64ch] vd:text-ui vd:leading-7 vd:text-mute">
            {post.dek}
          </p>
          <p className="vd:mt-2 vd:flex vd:flex-wrap vd:gap-x-4 vd:text-meta vd:text-faint">
            <span style={{ color: "var(--v-accent)" }}>
              {CATEGORY_LABEL[post.category]}
            </span>
            <span>{post.author}</span>
            <span className="v-tabular">{post.minutes} min</span>
            {post.updated && (
              <span className="v-tabular">updated {post.updated}</span>
            )}
            <span className="v-tabular vd:sm:hidden">{post.date}</span>
          </p>
        </div>
        {withMedia && (
          <span className="vd:hidden vd:h-24 vd:w-20 vd:shrink-0 vd:overflow-hidden vd:rounded-plate vd:sm:block">
            {post.image && (
              <Image
                src={post.image}
                alt=""
                sizes="80px"
                className="vd:size-full vd:object-cover vd:opacity-60 vd:saturate-50 vd:transition vd:duration-300 vd:group-hover:opacity-100 vd:group-hover:saturate-100 vd:motion-reduce:transition-none"
              />
            )}
          </span>
        )}
      </a>
      <Rule className="vd:opacity-60" />
    </li>
  );
}

export function LedgerBlog({ posts }: { posts: Post[] }) {
  const counts = posts.reduce<Record<string, number>>((all, post) => {
    all[post.category] = (all[post.category] ?? 0) + 1;
    return all;
  }, {});
  const withMedia = posts.some((post) => post.image);

  return (
    <div id="top">
      <SiteHead direction="ledger" section="Publication ledger" />

      <section className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:pt-16 vd:md:px-10">
        <div className="vd:grid vd:gap-10 vd:lg:grid-cols-12">
          <div className="vd:lg:col-span-7">
            <Label>The Illarin record</Label>
            <h1 className="vd:mt-5 vd:font-display vd:text-display vd:font-medium">
              Everything published, in the order it happened
            </h1>
            <p className="v-lede vd:mt-5 vd:font-prose vd:text-lede vd:text-mute">
              Announcements, release notes and longer writing, each kept with
              the date it went out and the edits made since.
            </p>
          </div>
          <dl className="vd:grid vd:grid-cols-3 vd:gap-6 vd:self-end vd:lg:col-span-4 vd:lg:col-start-9">
            {[
              { label: "Entries", value: posts.length },
              {
                label: "Categories",
                value: ORDER.filter((c) => counts[c]).length,
              },
              {
                label: "Authors",
                value: new Set(posts.map((p) => p.author)).size,
              },
            ].map((stat) => (
              <div key={stat.label}>
                <dd className="vd:font-display vd:text-[2.75rem] vd:leading-none vd:font-medium">
                  <Counter value={stat.value} />
                </dd>
                <dt className="vd:mt-2 vd:text-label vd:font-bold vd:uppercase vd:tracking-[0.16em] vd:text-mute">
                  {stat.label}
                </dt>
              </div>
            ))}
          </dl>
        </div>

        <nav className="vd:mt-group vd:flex vd:flex-wrap vd:gap-x-7 vd:gap-y-3">
          {ORDER.map((category) => (
            <a
              key={category}
              href="#top"
              className={cn(
                "vd:v-hit vd:flex vd:min-h-9 vd:items-center vd:gap-2.5 vd:text-meta vd:text-mute vd:hover:text-ink",
                !counts[category] && "vd:opacity-40",
              )}
            >
              <span
                aria-hidden="true"
                className="vd:size-2.5 vd:rounded-full"
                style={{ background: CATEGORY_PIGMENT[category] }}
              />
              {CATEGORY_LABEL[category]}
              <span className="v-tabular vd:text-faint">
                {counts[category] ?? 0}
              </span>
            </a>
          ))}
        </nav>
      </section>

      <section className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:pt-section vd:md:px-10">
        <div className="vd:flex vd:items-end vd:gap-6">
          <h2 className="v-tabular vd:font-display vd:text-[clamp(2.5rem,4vw,3.5rem)] vd:leading-[0.9] vd:font-medium">
            2026
          </h2>
          <span className="vd:mb-2.5 vd:flex-1">
            <Rule />
          </span>
          <p className="vd:mb-1 vd:text-meta vd:text-mute">
            {posts.length} entries
          </p>
        </div>
        {posts.length ? (
          <ul className="vd:mt-4">
            {posts.map((post, index) => (
              <Entry
                key={post.id}
                post={post}
                index={posts.length - index}
                withMedia={withMedia}
              />
            ))}
          </ul>
        ) : (
          <Empty
            title="Nothing recorded yet"
            detail="The first entry will open this year."
          />
        )}
      </section>

      <SiteFoot />
    </div>
  );
}
