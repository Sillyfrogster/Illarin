"use client";

import { ChevronDown, Download } from "lucide-react";
import Image from "next/image";
import darkQuiet from "@/assets/art/full/illarin-quiet-page-character-dark-v1.webp";
import lightQuiet from "@/assets/art/full/illarin-quiet-page-character-light-v1.webp";
import type { Asset } from "../assets";
import { KIND_LABEL } from "../assets";
import { BlockView } from "../block";
import { Formats } from "../formats";
import { History } from "../history";
import { Room } from "../motion";
import { SiteFoot, SiteHead } from "../shell";
import { Action, cn, Empty, Label, Tag, tintStyle } from "../ui";

function FormatPicker({
  formats,
  recommended,
  tone,
}: {
  formats: Asset["formats"];
  recommended?: string;
  tone: "ink" | "over";
}) {
  return (
    <details className="vd:group vd:relative">
      <summary
        className={cn(
          "vd:inline-flex vd:min-h-13 vd:cursor-pointer vd:list-none vd:items-center vd:gap-2 vd:rounded-control vd:px-5 vd:text-ui vd:font-semibold",
          tone === "over"
            ? "vd:text-over vd:shadow-[inset_0_0_0_1px_rgb(255_255_255/0.28)]"
            : "vd:text-ink vd:shadow-[inset_0_0_0_1px_var(--v-rule)]",
        )}
      >
        {recommended}
        <ChevronDown className="vd:size-4 vd:transition vd:group-open:rotate-180 vd:motion-reduce:transition-none" />
      </summary>
      <ul className="vd:mt-3 vd:grid vd:gap-1 vd:rounded-plate vd:bg-plane vd:p-2 vd:shadow-[inset_0_0_0_1px_var(--v-rule),0_24px_50px_-24px_rgb(0_0_0/0.4)] vd:sm:absolute vd:sm:z-20 vd:sm:w-88">
        {formats.map((format) => (
          <li key={format.label}>
            <a
              href="#top"
              className="vd:block vd:rounded-control vd:px-3 vd:py-2.5 vd:hover:bg-ink/6"
            >
              <span className="vd:block vd:text-ui vd:font-semibold">
                {format.label}
              </span>
              <span className="vd:block vd:text-meta vd:text-mute">
                {format.drops
                  ? `Leaves ${format.drops.length} things behind`
                  : format.note}
              </span>
            </a>
          </li>
        ))}
      </ul>
    </details>
  );
}

export function AmbientAsset({
  asset,
  theme,
}: {
  asset: Asset;
  theme: "light" | "dark";
}) {
  const dark = theme === "dark";
  const recommended = asset.formats.find((format) => format.recommended);
  const immersive = Boolean(asset.cover) && dark;

  return (
    <div id="top" style={tintStyle(asset.tint, dark)}>
      <SiteHead direction="ambient" section="Catalog" over={immersive} />

      {immersive && asset.cover ? (
        <section className="vd:relative vd:-mt-[4.5rem] vd:flex vd:min-h-[86dvh] vd:items-end vd:overflow-hidden">
          <Image
            src={asset.cover}
            alt=""
            priority
            sizes="100vw"
            aria-hidden="true"
            className="vd:absolute vd:inset-0 vd:size-full vd:scale-125 vd:object-cover vd:blur-3xl vd:saturate-150"
          />
          <div className="v-scrim vd:absolute vd:inset-0" />
          <div
            aria-hidden="true"
            className="vd:absolute vd:inset-x-0 vd:bottom-0 vd:h-40 vd:bg-linear-to-t vd:from-field vd:to-transparent"
          />
          <div className="vd:relative vd:mx-auto vd:grid vd:w-full vd:max-w-[86rem] vd:items-end vd:gap-x-14 vd:gap-y-8 vd:px-5 vd:pt-28 vd:pb-12 vd:md:px-10 vd:md:pb-16 vd:lg:grid-cols-12">
            <figure className="vd:order-2 vd:w-48 vd:sm:w-64 vd:lg:order-none vd:lg:col-span-4 vd:lg:w-full">
              <Image
                src={asset.cover}
                alt={`Cover of ${asset.name}`}
                priority
                sizes="(max-width: 64rem) 16rem, 26rem"
                className="vd:h-auto vd:w-full vd:rounded-plate vd:shadow-[0_40px_90px_-40px_rgb(0_0_0/0.9),inset_0_0_0_1px_rgb(255_255_255/0.14)]"
              />
            </figure>
            <div className="vd:lg:col-span-7 vd:lg:col-start-6">
              <Label className="vd:text-over-mute">
                {KIND_LABEL[asset.kind]} · Version {asset.assetVersion} · No
                adult content
              </Label>
              <h1 className="vd:mt-5 vd:max-w-[14ch] vd:font-display vd:text-hero vd:font-medium vd:text-over">
                {asset.name}
              </h1>
              <p className="v-lede vd:mt-6 vd:font-prose vd:text-lede vd:text-over-mute">
                {asset.blurb}
              </p>
              <div className="vd:mt-8 vd:flex vd:flex-wrap vd:items-center vd:gap-3">
                <Action variant="tinted" size="large">
                  <Download />
                  Download
                </Action>
                <FormatPicker
                  formats={asset.formats}
                  recommended={recommended?.label}
                  tone="over"
                />
              </div>
              <p className="vd:mt-6 vd:text-meta vd:text-over-mute">
                By{" "}
                <span className="vd:font-semibold vd:text-over">
                  {asset.creditedAuthor}
                </span>{" "}
                {asset.creator} · Shared {asset.shared}
              </p>
            </div>
          </div>
        </section>
      ) : (
        <Room className="vd:px-5 vd:pt-14 vd:md:px-10 vd:md:pt-20">
          <div className="vd:mx-auto vd:grid vd:max-w-[86rem] vd:items-center vd:gap-x-14 vd:gap-y-10 vd:lg:grid-cols-12">
            <figure className="vd:w-52 vd:sm:w-72 vd:lg:col-span-5 vd:lg:w-full">
              <Image
                src={asset.cover ?? (dark ? darkQuiet : lightQuiet)}
                alt={asset.cover ? `Cover of ${asset.name}` : ""}
                priority
                sizes="(max-width: 64rem) 18rem, 34rem"
                className="vd:h-auto vd:w-full vd:rounded-plate"
                style={{ boxShadow: "var(--v-ambient)" }}
              />
              {!asset.cover && (
                <figcaption className="vd:mt-3 vd:text-meta vd:text-faint">
                  No cover uploaded.
                </figcaption>
              )}
            </figure>
            <div className="vd:lg:col-span-7">
              <Label style={{ color: "var(--v-tint-ink)" }}>
                {KIND_LABEL[asset.kind]} · Version {asset.assetVersion} · No
                adult content
              </Label>
              <h1 className="vd:mt-5 vd:max-w-[14ch] vd:font-display vd:text-hero vd:font-medium">
                {asset.name}
              </h1>
              {asset.blurb ? (
                <p className="v-lede vd:mt-6 vd:font-prose vd:text-lede vd:text-mute">
                  {asset.blurb}
                </p>
              ) : (
                <p className="v-lede vd:mt-6 vd:font-prose vd:text-lede vd:text-faint vd:italic">
                  No blurb yet
                </p>
              )}
              <div className="vd:mt-8 vd:flex vd:flex-wrap vd:items-center vd:gap-3">
                <Action variant="tinted" size="large">
                  <Download />
                  Download
                </Action>
                <FormatPicker
                  formats={asset.formats}
                  recommended={recommended?.label}
                  tone="ink"
                />
              </div>
              <p className="vd:mt-6 vd:text-meta vd:text-mute">
                By{" "}
                <span className="vd:font-semibold vd:text-ink">
                  {asset.creditedAuthor}
                </span>{" "}
                {asset.creator} · Shared {asset.shared}
              </p>
              {Boolean(asset.tags.length) && (
                <ul className="vd:mt-6 vd:flex vd:flex-wrap vd:gap-2">
                  {asset.tags.map((tag) => (
                    <li key={tag}>
                      <Tag>{tag}</Tag>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </Room>
      )}

      <Room className="vd:px-5 vd:pt-section vd:md:px-10">
        <div className="vd:mx-auto vd:grid vd:max-w-[86rem] vd:gap-x-14 vd:gap-y-section vd:lg:grid-cols-12">
          {asset.blocks.map((block) => (
            <BlockView key={block.id} block={block} direction="ambient" />
          ))}

          <div className="vd:lg:col-span-12">
            <Formats formats={asset.formats} direction="ambient" />
          </div>

          <div className="vd:lg:col-span-12">
            {asset.releases.length ? (
              <History releases={asset.releases} direction="ambient" />
            ) : (
              <Empty
                title="No updates yet"
                detail="The first published update will open this record."
              />
            )}
          </div>
        </div>
      </Room>

      <SiteFoot />
    </div>
  );
}
