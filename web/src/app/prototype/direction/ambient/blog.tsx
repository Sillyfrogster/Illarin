"use client";

import Image from "next/image";
import { CATEGORY_LABEL, DARK_TINT, type Post } from "../data";
import { Arrive, Room } from "../motion";
import { More, SiteFoot, SiteHead } from "../shell";
import { Byline, cn, Empty, Label, tintStyle } from "../ui";

/** A repeating tile pattern, so rows always close and no two rows are shaped alike */
const TILES = [
  { cols: "vd:sm:col-span-3", rows: "vd:sm:row-span-3", wide: true },
  { cols: "vd:sm:col-span-3", rows: "vd:sm:row-span-3", wide: true },
  { cols: "vd:sm:col-span-2", rows: "vd:sm:row-span-2", wide: false },
  { cols: "vd:sm:col-span-2", rows: "vd:sm:row-span-2", wide: false },
  { cols: "vd:sm:col-span-2", rows: "vd:sm:row-span-2", wide: false },
  { cols: "vd:sm:col-span-4", rows: "vd:sm:row-span-3", wide: true },
  { cols: "vd:sm:col-span-2", rows: "vd:sm:row-span-3", wide: false },
];

function Plate({
  post,
  wide,
  dark,
}: {
  post: Post;
  wide: boolean;
  dark: boolean;
}) {
  return (
    <a
      href="#top"
      style={tintStyle(post.tint, dark)}
      className="vd:group vd:relative vd:block vd:size-full vd:min-h-64 vd:overflow-hidden vd:rounded-plate"
    >
      {post.image ? (
        <>
          <Image
            src={post.image}
            alt=""
            sizes="(max-width: 40rem) 100vw, 30rem"
            className="vd:absolute vd:inset-0 vd:size-full vd:object-cover vd:transition vd:duration-700 vd:group-hover:scale-[1.03] vd:motion-reduce:transition-none"
          />
          <div className="v-scrim vd:absolute vd:inset-0" />
        </>
      ) : (
        <>
          <div
            className="vd:absolute vd:inset-0"
            style={{
              background:
                "linear-gradient(158deg, color-mix(in oklab, var(--v-tint-a) 42%, var(--v-plane)) 0%, var(--v-plane) 76%)",
            }}
          />
          <p
            aria-hidden="true"
            className="vd:absolute vd:top-5 vd:right-5 vd:font-display vd:text-[5rem] vd:leading-none vd:font-medium vd:text-ink/10"
          >
            {CATEGORY_LABEL[post.category].slice(0, 1)}
          </p>
        </>
      )}
      <div className="vd:absolute vd:inset-0 vd:flex vd:flex-col vd:justify-end vd:gap-2.5 vd:overflow-hidden vd:p-6 vd:sm:p-7">
        <Label
          className={post.image ? "vd:text-over-mute" : undefined}
          style={post.image ? undefined : { color: "var(--v-tint-ink)" }}
        >
          {CATEGORY_LABEL[post.category]}
        </Label>
        <h3
          className={cn(
            "vd:line-clamp-3 vd:font-display vd:font-medium vd:tracking-[-0.012em] vd:wrap-pretty",
            post.image ? "vd:text-over" : "vd:text-ink",
            wide
              ? "vd:text-[clamp(1.375rem,1.9vw,1.75rem)] vd:leading-[1.16]"
              : "vd:text-[1.1875rem] vd:leading-[1.22]",
          )}
        >
          {post.title}
        </h3>
        {wide && (
          <p
            className={cn(
              "vd:line-clamp-2 vd:max-w-[48ch] vd:text-meta vd:leading-6",
              post.image ? "vd:text-over-mute" : "vd:text-mute",
            )}
          >
            {post.dek}
          </p>
        )}
        <Byline
          author={post.author}
          category={post.category}
          date={post.date}
          minutes={post.minutes}
          tone={post.image ? "over" : "mute"}
        />
      </div>
    </a>
  );
}

export function AmbientBlog({
  posts,
  theme,
}: {
  posts: Post[];
  theme: "light" | "dark";
}) {
  const dark = theme === "dark";
  const [feature, ...rest] = posts;
  const few = rest.length <= 3;

  return (
    <div id="top" style={tintStyle(feature?.tint ?? DARK_TINT, dark)}>
      <SiteHead
        direction="ambient"
        section="Journal"
        over={Boolean(feature?.image)}
      />

      {feature ? (
        <section
          className={cn(
            "vd:relative vd:-mt-[4.5rem] vd:flex vd:items-end vd:overflow-hidden",
            feature.image
              ? "vd:min-h-[78dvh]"
              : "v-room vd:min-h-[62dvh] vd:pt-24",
          )}
        >
          {feature.image && (
            <>
              <Image
                src={feature.image}
                alt=""
                priority
                sizes="100vw"
                className="vd:absolute vd:inset-0 vd:size-full vd:object-cover"
              />
              <div className="v-scrim vd:absolute vd:inset-0" />
              <div
                aria-hidden="true"
                className="vd:absolute vd:inset-x-0 vd:bottom-0 vd:h-40 vd:bg-linear-to-t vd:from-field vd:to-transparent"
              />
            </>
          )}
          <div className="vd:relative vd:mx-auto vd:w-full vd:max-w-[86rem] vd:px-5 vd:pt-24 vd:pb-14 vd:md:px-10 vd:md:pb-20">
            <Label
              className={feature.image ? "vd:text-over-mute" : undefined}
              style={feature.image ? undefined : { color: "var(--v-tint-ink)" }}
            >
              {CATEGORY_LABEL[feature.category]}
            </Label>
            <h1
              className={cn(
                "vd:mt-5 vd:max-w-[17ch] vd:font-display vd:text-hero vd:font-medium",
                feature.image && "vd:text-over",
              )}
            >
              {feature.title}
            </h1>
            <p
              className={cn(
                "v-lede vd:mt-6 vd:font-prose vd:text-lede",
                feature.image ? "vd:text-over-mute" : "vd:text-mute",
              )}
            >
              {feature.dek}
            </p>
            <div className="vd:mt-7 vd:flex vd:flex-wrap vd:items-center vd:gap-x-8 vd:gap-y-4">
              <Byline
                author={feature.author}
                handle={feature.handle}
                category={feature.category}
                date={feature.date}
                minutes={feature.minutes}
                tone={feature.image ? "over" : "mute"}
              />
              <More tone={feature.image ? "over" : "ink"}>Read this</More>
            </div>
          </div>
        </section>
      ) : (
        <Room className="vd:px-5 vd:pt-24 vd:md:px-10">
          <div className="vd:mx-auto vd:max-w-[86rem]">
            <Empty
              title="Nothing published yet"
              detail="The first announcement will appear here."
            />
          </div>
        </Room>
      )}

      <Room className="vd:px-5 vd:pt-section vd:md:px-10">
        <div className="vd:mx-auto vd:max-w-[86rem]">
          <div className="vd:flex vd:flex-wrap vd:items-baseline vd:justify-between vd:gap-4">
            <h2 className="vd:font-display vd:text-display vd:font-medium">
              Everything else
            </h2>
            <p className="vd:text-meta vd:text-mute">
              {rest.length} more from 2026
            </p>
          </div>
          {rest.length ? (
            <div className="vd:mt-group vd:grid vd:gap-4 vd:sm:auto-rows-[8.5rem] vd:sm:grid-cols-6 vd:sm:gap-5">
              {rest.map((post, index) => {
                const tile = few ? TILES[0] : TILES[index % TILES.length];
                return (
                  <Arrive
                    key={post.id}
                    distance={16}
                    className={cn(tile.cols, tile.rows)}
                  >
                    <Plate post={post} wide={tile.wide} dark={dark} />
                  </Arrive>
                );
              })}
            </div>
          ) : (
            <Empty
              title="One entry so far"
              detail="Everything else published will collect here."
            />
          )}
        </div>
        <SiteFoot />
      </Room>
    </div>
  );
}
