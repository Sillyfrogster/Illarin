"use client";

import { Download } from "lucide-react";
import Image from "next/image";
import type { CSSProperties } from "react";
import darkQuiet from "@/assets/art/full/illarin-quiet-page-character-dark-v1.webp";
import lightQuiet from "@/assets/art/full/illarin-quiet-page-character-light-v1.webp";
import type { Asset } from "../assets";
import { KIND_LABEL } from "../assets";
import { BlockView } from "../block";
import { Formats } from "../formats";
import { History } from "../history";
import { SiteFoot, SiteHead } from "../shell";
import { Action, Empty, Label, Rule, Tag } from "../ui";

export function LedgerAsset({
  asset,
  theme,
}: {
  asset: Asset;
  theme: "light" | "dark";
}) {
  const specification: [string, string][] = [
    ["Kind", KIND_LABEL[asset.kind]],
    ["Version", asset.assetVersion],
    ["Credited author", asset.creditedAuthor],
    ["Account", asset.creator],
    ["Shared", asset.shared],
    ["Sections", String(asset.blocks.length)],
    ["Recorded versions", String(asset.releases.length)],
    ["Adult content", "No"],
  ];

  return (
    <div id="top" style={{ "--v-accent": "var(--v-indigo)" } as CSSProperties}>
      <SiteHead direction="ledger" section="Catalog record" />

      <div className="vd:mx-auto vd:max-w-[86rem] vd:px-5 vd:pt-14 vd:md:px-10">
        <header className="vd:grid vd:gap-x-12 vd:gap-y-10 vd:lg:grid-cols-12">
          <div className="vd:lg:col-span-5">
            <Image
              src={asset.cover ?? (theme === "light" ? lightQuiet : darkQuiet)}
              alt={asset.cover ? `Cover of ${asset.name}` : ""}
              priority
              sizes="(max-width: 64rem) 100vw, 28rem"
              className="vd:h-auto vd:w-full vd:rounded-plate vd:shadow-[inset_0_0_0_1px_var(--v-rule)]"
            />
            {!asset.cover && (
              <p className="vd:mt-3 vd:text-meta vd:text-faint">
                No cover uploaded. Illarin shows its own plate for a{" "}
                {KIND_LABEL[asset.kind].toLowerCase()}.
              </p>
            )}
          </div>

          <div className="vd:lg:col-span-7">
            <Label style={{ color: "var(--v-accent)" }}>Catalog record</Label>
            <h1 className="vd:mt-4 vd:max-w-[16ch] vd:font-display vd:text-display vd:font-medium">
              {asset.name}
            </h1>
            {asset.blurb ? (
              <p className="v-lede vd:mt-5 vd:font-prose vd:text-lede vd:text-mute">
                {asset.blurb}
              </p>
            ) : (
              <p className="v-lede vd:mt-5 vd:font-prose vd:text-lede vd:text-faint vd:italic">
                No blurb yet
              </p>
            )}

            <dl className="v-tabular vd:mt-8 vd:grid vd:gap-x-12 vd:sm:grid-cols-2">
              {specification.map(([term, value]) => (
                <div
                  key={term}
                  className="vd:flex vd:items-baseline vd:gap-3 vd:py-2.5 vd:shadow-[inset_0_-1px_0_var(--v-hair)]"
                >
                  <dt className="vd:shrink-0 vd:text-meta vd:text-mute">
                    {term}
                  </dt>
                  <span
                    aria-hidden="true"
                    className="v-leader vd:mb-[0.4em] vd:h-px vd:min-w-4 vd:flex-1 vd:self-end"
                  />
                  <dd className="vd:text-meta vd:font-semibold">{value}</dd>
                </div>
              ))}
            </dl>

            {Boolean(asset.tags.length) && (
              <ul className="vd:mt-6 vd:flex vd:flex-wrap vd:gap-2">
                {asset.tags.map((tag) => (
                  <li key={tag}>
                    <Tag>{tag}</Tag>
                  </li>
                ))}
              </ul>
            )}

            <div className="vd:mt-8 vd:flex vd:flex-wrap vd:gap-3">
              <Action variant="primary" size="large">
                <Download />
                Download
              </Action>
              <Action variant="secondary" size="large">
                Choose a format
              </Action>
            </div>
          </div>
        </header>

        <div className="vd:mt-chapter vd:grid vd:gap-x-12 vd:gap-y-section vd:lg:grid-cols-12">
          {asset.blocks.map((block, index) => (
            <BlockView
              key={block.id}
              block={block}
              direction="ledger"
              number={index + 1}
            />
          ))}

          <div className="vd:lg:col-span-12">
            <Rule />
            <div className="vd:h-6" />
            <Formats
              formats={asset.formats}
              direction="ledger"
              number={asset.blocks.length + 1}
            />
          </div>

          <div className="vd:lg:col-span-12">
            {asset.releases.length ? (
              <History
                releases={asset.releases}
                direction="ledger"
                number={asset.blocks.length + 2}
              />
            ) : (
              <Empty
                title="No updates recorded"
                detail="This record opens when the first update is published."
              />
            )}
          </div>
        </div>
      </div>

      <SiteFoot />
    </div>
  );
}
