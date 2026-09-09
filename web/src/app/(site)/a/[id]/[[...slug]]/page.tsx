import { ArrowLeft } from "lucide-react";
import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { cache } from "react";
import { ChipSet } from "@/components/ui/Chip";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import { type AssetDetail, fetchAsset } from "@/lib/api/query";
import { assetMetadata } from "@/lib/asset-metadata";
import { assetDisplayName } from "@/lib/asset-name";
import { assetHoldsNothing } from "@/lib/asset-page-content";
import { assetRedirect, isAssetId } from "@/lib/asset-url";
import { cn } from "@/lib/cn";
import { KIND_LABELS } from "@/lib/kinds";
import { protectedAppLabel } from "@/lib/protected-apps";
import { formattingWasRemoved } from "@/lib/rich-text";
import { WorkingCopyProvider } from "@/lib/working-copy";
import { AssetBlocks } from "./AssetBlocks";
import { AssetMedia } from "./AssetMedia";
import { DraftHeaderActions } from "./DraftHeaderActions";
import { GetAsset } from "./GetAsset";
import { LatestUpdate } from "./LatestUpdate";
import { UpdatePanel } from "./UpdatePanel";
import { WithholdNotice } from "./WithholdNotice";
import { WorkingCopyNotice } from "./WorkingCopyNotice";

const loadAsset = cache(async (id: string): Promise<AssetDetail | null> => {
  if (!isAssetId(id)) return null;
  const cookie = (await cookies()).toString();
  return fetchAsset(id, cookie);
});

const SHELL = "mx-auto w-full max-w-[var(--shell)] px-[var(--gutter)]";

const TAG_PREVIEW_LIMIT = 8;

function ratingLabel(isNsfw: boolean | null): string {
  if (isNsfw === null) return "Rating not set";
  return isNsfw ? "Adult content" : "No adult content";
}

function browseTagHref(value: string): string {
  const quoted = value.includes(" ") ? `"${value}"` : value;
  return `/browse?q=${encodeURIComponent(`tag:${quoted}`)}`;
}

export async function generateMetadata({
  params,
}: PageProps<"/a/[id]/[[...slug]]">): Promise<Metadata> {
  const asset = await loadAsset((await params).id);
  return asset ? assetMetadata(asset) : { title: "Not found" };
}

export default async function AssetPage({
  params,
}: PageProps<"/a/[id]/[[...slug]]">) {
  const { id, slug } = await params;
  const published = await loadAsset(id);
  if (!published) notFound();

  const canonical = assetRedirect({ id, slug }, published);
  if (canonical) redirect(canonical);

  const asset = published.isOwner
    ? await fetchAsset(id, (await cookies()).toString(), true)
    : published;
  if (!asset) notFound();

  const kind = KIND_LABELS[asset.kind];
  const isDraft = asset.lifecycle === "draft";
  const sharedDate = new Date(asset.createdAt).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
  const holdsNothing = assetHoldsNothing(asset.blocks);
  const updatable = asset.isOwner && !isDraft && !asset.withhold;
  const draftable = isDraft && asset.isOwner && Boolean(asset.readiness);

  return (
    <WorkingCopyProvider key={asset.id} version={asset.workingCopyVersion}>
      <div className="relative isolate overflow-x-clip pb-chapter">
        {asset.isOwner ? <WorkingCopyNotice /> : null}
        <article>
          <div className={SHELL}>
            <Link
              className="mt-6 inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
              href="/browse"
            >
              <ArrowLeft aria-hidden="true" className="size-4" />
              Back to the collection
            </Link>

            {draftable || updatable ? (
              <div className="mt-4 flex flex-wrap items-stretch gap-4 [&>*]:w-full [&>*]:max-w-md">
                {draftable ? <DraftHeaderActions /> : null}
                {updatable ? (
                  <UpdatePanel
                    assetId={asset.id}
                    kind={kind.toLowerCase()}
                    unpublishedChanges={Boolean(asset.unpublishedChanges)}
                  />
                ) : null}
              </div>
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
                <h1
                  className={cn(
                    "max-w-[15ch] font-display text-hero font-medium tracking-[-0.035em] break-words",
                    asset.name ? "text-ink" : "text-mute italic",
                  )}
                >
                  {assetDisplayName(asset.name)}
                </h1>
                <p className="mt-5 flex flex-wrap items-center gap-x-2 gap-y-1 text-ui text-mute">
                  <span
                    aria-hidden="true"
                    className="size-2 shrink-0 rounded-full bg-accent"
                  />
                  {kind}
                  <span aria-hidden="true">·</span>
                  {ratingLabel(asset.isNsfw)}
                  {asset.linkedInstallOnly ? (
                    <>
                      <span aria-hidden="true">·</span>
                      Linked install only
                    </>
                  ) : null}
                </p>
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
                    The creator has not written a blurb for this{" "}
                    {kind.toLowerCase()} yet.
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
                    This {kind.toLowerCase()} installs only through a linked
                    app. Allowed apps:{" "}
                    {asset.allowedApps.map(protectedAppLabel).join(", ")}.
                  </p>
                ) : null}

                {asset.withhold ? (
                  <WithholdNotice withhold={asset.withhold} />
                ) : null}

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

          <AssetBlocks
            addableBlocks={asset.addableBlocks ?? []}
            allowedApps={asset.allowedApps}
            assetId={asset.id}
            blocks={asset.blocks}
            creatorMenu={{
              assetId: asset.id,
              creator: asset.creator,
              discovery: asset.discovery,
              hasOriginal: Boolean(asset.original),
              isDraft,
              isNsfw: asset.isNsfw,
              isOwner: asset.isOwner,
              kind: kind.toLowerCase(),
              name: asset.name,
              readiness: asset.readiness,
              sealedBlocks: asset.sealedBlocks,
              sealsPrompts: asset.linkedInstallOnly,
              withheld: Boolean(asset.withhold),
            }}
            eligibleApps={asset.eligibleApps}
            images={asset.media}
            isOwner={asset.isOwner}
            kind={asset.kind}
            shellClassName={SHELL}
          />
        </article>
      </div>
    </WorkingCopyProvider>
  );
}
