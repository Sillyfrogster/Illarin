"use client";

import Image from "next/image";
import { useState } from "react";
import darkMasthead from "@/assets/art/full/illarin-blog-masthead-dark-v1.webp";
import lightMasthead from "@/assets/art/full/illarin-blog-masthead-light-v1.webp";
import { CATEGORY_LABEL, type Category, type Post } from "../data";
import { EdgeBeam, RiseText, SpineMedia } from "../motion";
import { More, SiteFoot, SiteHead } from "../shell";
import { Byline, cn, Empty, Label, Rule } from "../ui";

function Entry({
  post,
  index,
  open,
  onOpen,
}: {
  post: Post;
  index: number;
  open: boolean;
  onOpen: () => void;
}) {
  return (
    <li>
      <a
        href="#top"
        data-open={open}
        onMouseEnter={onOpen}
        onFocus={onOpen}
        className="v-row v-row-shift vd:group vd:block vd:py-8"
      >
        <div className="vd:flex vd:gap-5 vd:sm:gap-8">
          <span className="v-tabular vd:hidden vd:w-7 vd:shrink-0 vd:pt-[0.55rem] vd:text-meta vd:text-faint vd:sm:block">
            {String(index).padStart(2, "0")}
          </span>
          <div className="vd:min-w-0 vd:flex-1">
            <h3 className="vd:font-display vd:text-[clamp(1.5rem,2.2vw,2.125rem)] vd:leading-[1.12] vd:font-medium vd:tracking-[-0.012em] vd:wrap-pretty">
              {post.title}
            </h3>
            <p className="vd:mt-3 vd:max-w-[58ch] vd:text-[1rem] vd:leading-7 vd:text-mute">
              {post.dek}
            </p>
            <Byline
              className="vd:mt-4"
              author={post.author}
              category={post.category}
              date={post.date}
              updated={post.updated}
              minutes={post.minutes}
            />
          </div>
        </div>
        <div className="vd:relative vd:mt-8">
          <Rule />
          <span className="v-row-rule v-accent-rule vd:absolute vd:inset-x-0 vd:top-0" />
        </div>
      </a>
    </li>
  );
}

export function VitrineBlog({
  posts,
  theme,
}: {
  posts: Post[];
  theme: "light" | "dark";
}) {
  const [lead, ...rest] = posts;
  const [openId, setOpenId] = useState(rest[0]?.id ?? lead?.id);
  const open = posts.find((post) => post.id === openId) ?? lead;
  const counts = posts.reduce<Record<string, number>>((all, post) => {
    all[post.category] = (all[post.category] ?? 0) + 1;
    return all;
  }, {});

  return (
    <div id="top">
      <SiteHead direction="vitrine" section="Journal" />

      <section className="vd:relative">
        <div className="vd:relative vd:h-[clamp(10rem,22vw,22rem)] vd:w-full vd:overflow-hidden">
          <Image
            src={theme === "light" ? lightMasthead : darkMasthead}
            alt=""
            priority
            sizes="100vw"
            className="vd:h-full vd:w-full vd:object-cover vd:object-[70%_center]"
          />
        </div>
        <Rule accent />
        <div className="vd:relative vd:mx-auto vd:grid vd:max-w-[86rem] vd:items-center vd:gap-x-14 vd:gap-y-10 vd:px-5 vd:pt-14 vd:pb-16 vd:md:px-10 vd:md:pt-20 vd:md:pb-24 vd:lg:grid-cols-12">
          {lead ? (
            <>
              <div className="vd:lg:col-span-7">
                <Label>The Illarin journal</Label>
                <h1 className="vd:mt-6 vd:max-w-[15ch] vd:font-display vd:text-hero vd:font-medium">
                  <RiseText text={lead.title} />
                </h1>
                <p className="v-lede vd:mt-7 vd:font-prose vd:text-lede vd:text-mute">
                  {lead.dek}
                </p>
                <Byline
                  className="vd:mt-7"
                  author={lead.author}
                  handle={lead.handle}
                  category={lead.category}
                  date={lead.date}
                  minutes={lead.minutes}
                />
                <More className="vd:mt-8">Read this dispatch</More>
              </div>
              <div className="vd:lg:col-span-5">
                {lead.image ? (
                  <figure className="vd:relative vd:overflow-hidden vd:rounded-plate">
                    <Image
                      src={lead.image}
                      alt=""
                      sizes="(max-width: 64rem) 100vw, 34rem"
                      className="vd:h-auto vd:w-full"
                    />
                    <EdgeBeam />
                  </figure>
                ) : (
                  <div className="vd:max-w-[34rem] vd:pt-2 vd:lg:pl-10 vd:lg:shadow-[inset_1px_0_0_var(--v-hair)]">
                    <Label>About this journal</Label>
                    <p className="vd:mt-4 vd:font-prose vd:text-[1.0625rem] vd:leading-8 vd:text-mute">
                      Release notes, announcements and longer writing from
                      Illarin and the projects it publishes for. Everything here
                      is written by a person and kept in the open.
                    </p>
                    <More className="vd:mt-6">Subscribe by feed</More>
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="vd:lg:col-span-8">
              <Label>The Illarin journal</Label>
              <Empty
                title="Nothing published yet"
                detail="The first announcement will appear here."
              />
            </div>
          )}
        </div>
      </section>

      <section className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:pt-section vd:md:px-10">
        <div className="vd:grid vd:gap-10 vd:lg:grid-cols-12 vd:lg:gap-14">
          <div className="vd:lg:col-span-3">
            <div className="vd:lg:sticky vd:lg:top-24">
              <p
                aria-hidden="true"
                className="v-tabular vd:font-display vd:text-[clamp(4rem,7vw,7.5rem)] vd:leading-[0.8] vd:font-medium vd:text-ink/8"
              >
                2026
              </p>
              <div className="vd:mt-8 vd:max-w-[17rem]">
                <SpineMedia
                  image={open?.image}
                  alt=""
                  fallback={
                    <>
                      <span
                        aria-hidden="true"
                        className="vd:absolute vd:inset-0"
                        style={{
                          background:
                            "linear-gradient(162deg, var(--v-accent-wash) 0%, transparent 72%)",
                        }}
                      />
                      <span className="v-accent-rule vd:absolute vd:inset-x-0 vd:top-0" />
                      <p className="vd:relative vd:font-display vd:text-[1.75rem] vd:leading-tight vd:font-medium vd:text-ink">
                        {open ? CATEGORY_LABEL[open.category] : ""}
                        <span className="vd:mt-2 vd:block vd:text-meta vd:font-normal vd:text-mute">
                          No picture with this entry
                        </span>
                      </p>
                    </>
                  }
                />
                <p className="vd:mt-3 vd:text-meta vd:text-faint">
                  {open
                    ? `${CATEGORY_LABEL[open.category]} · ${open.date}`
                    : "No entries"}
                </p>
              </div>
              <nav className="vd:mt-10 vd:max-w-[17rem]">
                <Label>Filter</Label>
                <ul className="vd:mt-3">
                  {(["announcement", "article", "release"] as Category[]).map(
                    (category) => (
                      <li key={category}>
                        <a
                          href="#top"
                          className={cn(
                            "vd:flex vd:min-h-11 vd:items-center vd:justify-between vd:gap-4 vd:text-ui vd:text-mute vd:shadow-[inset_0_-1px_0_var(--v-hair)] vd:hover:text-ink",
                            !counts[category] && "vd:opacity-40",
                          )}
                        >
                          {CATEGORY_LABEL[category]}
                          <span className="v-tabular vd:text-meta vd:text-faint">
                            {counts[category] ?? 0}
                          </span>
                        </a>
                      </li>
                    ),
                  )}
                </ul>
              </nav>
            </div>
          </div>

          <div className="vd:lg:col-span-8 vd:lg:col-start-5">
            <Rule />
            {rest.length ? (
              <ul>
                {rest.map((post, index) => (
                  <Entry
                    key={post.id}
                    post={post}
                    index={index + 2}
                    open={post.id === openId}
                    onOpen={() => setOpenId(post.id)}
                  />
                ))}
              </ul>
            ) : (
              <Empty
                title="One entry so far"
                detail="Everything else published will collect under this year."
              />
            )}
          </div>
        </div>
      </section>

      <SiteFoot />
    </div>
  );
}
