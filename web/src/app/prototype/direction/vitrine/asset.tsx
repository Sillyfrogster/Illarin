"use client";

import { Download, Link2, Share2 } from "lucide-react";
import Image from "next/image";
import darkArt from "@/assets/art/full/illarin-detail-page-art-dark-v2.webp";
import lightArt from "@/assets/art/full/illarin-detail-page-art-light-v2.webp";
import darkQuiet from "@/assets/art/full/illarin-quiet-page-character-dark-v1.webp";
import lightQuiet from "@/assets/art/full/illarin-quiet-page-character-light-v1.webp";
import type { Asset } from "../assets";
import { KIND_LABEL } from "../assets";
import { BlockView } from "../block";
import { Formats } from "../formats";
import { History } from "../history";
import { Dock, EdgeBeam, RiseText } from "../motion";
import { SiteFoot, SiteHead } from "../shell";
import { Action, Empty, Label, Rule, Tag } from "../ui";

export function VitrineAsset({
  asset,
  theme,
}: {
  asset: Asset;
  theme: "light" | "dark";
}) {
  const light = theme === "light";
  const recommended = asset.formats.find((format) => format.recommended);

  return (
    <div id="top">
      <SiteHead direction="vitrine" section="Catalog" />

      <section className="vd:relative vd:overflow-hidden">
        <Image
          src={light ? lightArt : darkArt}
          alt=""
          priority
          sizes="(max-width: 60rem) 100vw, 70rem"
          style={{
            maskImage:
              "linear-gradient(to left, black 12%, transparent 88%), linear-gradient(to top, transparent 0%, black 26%)",
            maskComposite: "intersect",
            WebkitMaskComposite: "source-in",
          }}
          className="vd:pointer-events-none vd:absolute vd:top-0 vd:right-[-12%] vd:h-full vd:w-[70%] vd:object-cover vd:opacity-90 vd:max-lg:opacity-12"
        />
        <div className="vd:relative vd:mx-auto vd:grid vd:max-w-[86rem] vd:items-end vd:gap-x-14 vd:gap-y-10 vd:px-5 vd:pt-12 vd:pb-14 vd:md:px-10 vd:md:pt-16 vd:lg:grid-cols-12">
          <div className="vd:lg:col-span-7">
            <Label>
              {KIND_LABEL[asset.kind]} · Version {asset.assetVersion} · No adult
              content
            </Label>
            <h1 className="vd:mt-6 vd:max-w-[13ch] vd:font-display vd:text-hero vd:font-medium">
              <RiseText text={asset.name} />
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

          <div className="vd:lg:col-span-4 vd:lg:col-start-9">
            <figure className="vd:relative vd:ml-auto vd:max-w-[22rem]">
              <span
                aria-hidden="true"
                className="vd:absolute vd:-inset-x-8 vd:-bottom-7 vd:h-20 vd:rounded-[50%] vd:blur-2xl"
                style={{ background: "var(--v-accent-line)", opacity: 0.5 }}
              />
              <span className="vd:relative vd:block vd:overflow-hidden vd:rounded-plate vd:shadow-[0_36px_70px_-30px_rgb(12_12_18/0.55)]">
                <Image
                  src={asset.cover ?? (light ? lightQuiet : darkQuiet)}
                  alt={asset.cover ? `Cover of ${asset.name}` : ""}
                  priority
                  sizes="(max-width: 64rem) 60vw, 26rem"
                  className="vd:h-auto vd:w-full"
                />
                <EdgeBeam seconds={14} />
              </span>
              {!asset.cover && (
                <figcaption className="vd:mt-3 vd:text-meta vd:text-faint">
                  No cover uploaded. Illarin shows its own plate for a{" "}
                  {KIND_LABEL[asset.kind].toLowerCase()}.
                </figcaption>
              )}
            </figure>
          </div>
        </div>
        <div className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:md:px-10">
          <Rule accent />
        </div>
      </section>

      <div className="vd:mx-auto vd:grid vd:max-w-[86rem] vd:gap-x-14 vd:gap-y-section vd:px-5 vd:pt-section vd:md:px-10 vd:lg:grid-cols-12">
        {asset.blocks.map((block) => (
          <BlockView key={block.id} block={block} direction="vitrine" />
        ))}

        <div className="vd:lg:col-span-12">
          <Formats formats={asset.formats} direction="vitrine" />
        </div>

        <div className="vd:lg:col-span-12">
          {asset.releases.length ? (
            <History releases={asset.releases} direction="vitrine" />
          ) : (
            <Empty
              title="No updates yet"
              detail="The first published update will open this record."
            />
          )}
        </div>
      </div>

      <SiteFoot />

      <Dock>
        <Action variant="primary" className="vd:gap-2">
          <Download />
          Download
        </Action>
        <span className="vd:hidden vd:px-3 vd:text-meta vd:whitespace-nowrap vd:text-mute vd:sm:block">
          {recommended?.label}
        </span>
        <Action variant="quiet" className="vd:whitespace-nowrap">
          {asset.formats.length} formats
        </Action>
        <Action variant="quiet" size="icon" aria-label="Copy link">
          <Link2 />
        </Action>
        <Action variant="quiet" size="icon" aria-label="Share">
          <Share2 />
        </Action>
      </Dock>
    </div>
  );
}
