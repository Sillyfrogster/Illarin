"use client";

import { ArrowLeft, PencilLine } from "lucide-react";
import Link from "next/link";
import { ChipSet } from "@/components/ui/Chip";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import type { AssetDetail } from "@/lib/api/query";
import { assetDisplayName } from "@/lib/asset-name";
import { cn } from "@/lib/cn";
import { protectedAppLabel } from "@/lib/protected-apps";
import { formattingWasRemoved } from "@/lib/rich-text";
import { AssetMedia } from "./AssetMedia";
import { GetAsset } from "./GetAsset";
import { LatestUpdate } from "./LatestUpdate";
import { WithholdNotice } from "./WithholdNotice";
import { EditableText } from "./workspace/EditableText";
import { useWorkspace } from "./workspace/state";

const TAG_PREVIEW_LIMIT = 8;

const RATINGS: { label: string; value: boolean | null }[] = [
  { label: "Not yet", value: null },
  { label: "No adult content", value: false },
  { label: "Adult content", value: true },
];

function ratingLabel(isNsfw: boolean | null): string {
  if (isNsfw === null) return "Rating not set";
  return isNsfw ? "Adult content" : "No adult content";
}

function browseTagHref(value: string): string {
  const quoted = value.includes(" ") ? `"${value}"` : value;
  return `/browse?q=${encodeURIComponent(`tag:${quoted}`)}`;
}

export function AssetHeader({
  asset,
  holdsNothing,
  kind,
  sharedDate,
  shellClassName,
}: {
  asset: AssetDetail;
  holdsNothing: boolean;
  kind: string;
  sharedDate: string;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const isDraft = asset.lifecycle === "draft";
  const writing = workspace.editing;
  const ratings = isDraft
    ? RATINGS
    : RATINGS.filter((rating) => rating.value !== null);

  return (
    <div className={shellClassName}>
      <div className="mt-6 flex flex-wrap items-center justify-between gap-4">
        <Link
          className="inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
          href="/browse"
        >
          <ArrowLeft aria-hidden="true" className="size-4" />
          Back to the collection
        </Link>
        {asset.isOwner && !writing ? (
          <button
            className="inline-flex min-h-11 items-center gap-2 rounded-control bg-action px-5 text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90"
            onClick={workspace.startEditing}
            type="button"
          >
            <PencilLine aria-hidden="true" size={16} />
            Edit this page
          </button>
        ) : null}
      </div>

      {asset.isOwner && writing ? (
        <p className="mt-4 text-meta text-mute">
          Click any writing on the page to edit it where it sits.
        </p>
      ) : null}

      <div
        className={cn(
          "grid items-center gap-8 py-8 lg:gap-12 lg:py-14",
          asset.media.length > 0
            ? "md:grid-cols-2 lg:grid-cols-[1fr_minmax(260px,0.9fr)_1fr]"
            : "md:grid-cols-[1.15fr_1fr]",
        )}
      >
        <div className="min-w-0">
          <EditableText
            active={workspace.cursor === "identity:name"}
            activate={() => workspace.setCursor("identity:name")}
            as="h1"
            className={cn(
              "max-w-[15ch] font-display text-hero font-medium tracking-[-0.035em] break-words",
              asset.name ? "text-ink" : "text-mute italic",
            )}
            done={() => workspace.setCursor(null)}
            id="asset-name"
            label="Name"
            live={writing}
            onChange={(name) =>
              workspace.writeIdentity({ ...workspace.identity, name })
            }
            placeholder="Name this page"
            singleLine
            value={
              writing
                ? workspace.identity.name
                : assetDisplayName(workspace.identity.name)
            }
          />
          <p className="mt-5 flex flex-wrap items-center gap-x-2 gap-y-1 text-ui text-mute">
            <span
              aria-hidden="true"
              className="size-2 shrink-0 rounded-full bg-accent"
            />
            {kind}
            {writing ? null : (
              <>
                <span aria-hidden="true">·</span>
                {ratingLabel(workspace.identity.isNsfw)}
              </>
            )}
            {asset.linkedInstallOnly ? (
              <>
                <span aria-hidden="true">·</span>
                Linked install only
              </>
            ) : null}
            {isDraft ? (
              <span className="rounded-control bg-accent-wash px-2 py-0.5 text-label font-medium text-ink">
                Private draft
              </span>
            ) : null}
          </p>
          {writing ? (
            <fieldset
              className="mt-4 min-w-0 border-0 p-0"
              id="adult-content-answer"
            >
              <legend className="text-label text-mute">Adult content</legend>
              <div className="mt-2 flex flex-wrap gap-2">
                {ratings.map((rating) => (
                  <button
                    aria-pressed={rating.value === workspace.identity.isNsfw}
                    className="min-h-11 rounded-control bg-deep px-4 text-meta text-ink outline-offset-3 aria-pressed:bg-action aria-pressed:text-on-accent"
                    key={rating.label}
                    onClick={() =>
                      workspace.writeIdentity({
                        ...workspace.identity,
                        isNsfw: rating.value,
                      })
                    }
                    type="button"
                  >
                    {rating.label}
                  </button>
                ))}
              </div>
              {workspace.identity.isNsfw === null ? (
                <p className="mt-2 text-label text-mute">
                  Publishing waits on this answer. Nothing answers it for you.
                </p>
              ) : null}
            </fieldset>
          ) : null}
          <p className="mt-7 text-ui text-mute">
            <Link
              className="font-medium text-ink underline decoration-accent/55 underline-offset-4 hover:decoration-accent"
              href={`/@${asset.creator}`}
            >
              {asset.creator}
            </Link>
            <span className="ml-2">
              {isDraft ? `Started ${sharedDate}` : `Shared ${sharedDate}`}
            </span>
          </p>
        </div>

        {asset.media.length > 0 ? (
          <div className="min-w-0 md:row-span-2 lg:row-span-1">
            <AssetMedia
              id={asset.id}
              isNsfw={asset.isNsfw}
              kind={asset.kind}
              media={asset.media}
              name={asset.name}
              visibility={asset.visibility}
            />
          </div>
        ) : null}

        <div className="min-w-0 md:col-start-1 lg:col-start-auto">
          {asset.blurb ? (
            <div className="max-w-[42ch] font-prose text-lede text-ink">
              <RichText text={asset.blurb} />
              {formattingWasRemoved([asset.blurb]) ? (
                <FormattingNotice />
              ) : null}
            </div>
          ) : (
            <p className="max-w-[42ch] font-prose text-lede text-mute">
              The creator has not written a blurb for this {kind.toLowerCase()}{" "}
              yet.
            </p>
          )}

          {asset.tags.length > 0 ? (
            <ChipSet
              className="mt-5 max-w-[42ch]"
              items={asset.tags.map((tag) => ({
                href: browseTagHref(tag.value),
                id: tag.value,
                label: tag.label,
              }))}
              limit={TAG_PREVIEW_LIMIT}
            />
          ) : null}

          {isDraft ? null : (
            <div className="mt-7">
              <GetAsset
                assetId={asset.id}
                blocks={asset.blocks}
                downloads={asset.downloads}
                holdsNothing={holdsNothing}
                images={asset.media}
                isOwner={asset.isOwner}
                kindLabel={kind.toLowerCase()}
                linkedInstallOnly={asset.linkedInstallOnly}
                original={asset.original}
              />
            </div>
          )}

          {asset.linkedInstallOnly ? (
            <p className="mt-4 max-w-[42ch] text-meta text-mute">
              This {kind.toLowerCase()} installs only through a linked app.
              Allowed apps:{" "}
              {asset.allowedApps.map(protectedAppLabel).join(", ")}.
            </p>
          ) : null}

          {asset.withhold ? <WithholdNotice withhold={asset.withhold} /> : null}

          {asset.latestUpdate ? (
            <LatestUpdate
              assetId={asset.id}
              kind={kind.toLowerCase()}
              version={asset.latestUpdate}
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}
