import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { cache } from "react";
import { type AssetDetail, fetchAsset } from "@/lib/api/query";
import { assetMetadata } from "@/lib/asset-metadata";
import { assetHoldsNothing } from "@/lib/asset-page-content";
import { assetRedirect, isAssetId } from "@/lib/asset-url";
import { KIND_LABELS } from "@/lib/kinds";
import { WorkingCopyProvider } from "@/lib/working-copy";
import { AssetBlocks } from "./AssetBlocks";
import { AssetHeader } from "./AssetHeader";
import { AssetWorkspace } from "./workspace/state";
import { WorkspaceSurfaces } from "./workspace/WorkspaceSurfaces";

const loadAsset = cache(async (id: string): Promise<AssetDetail | null> => {
  if (!isAssetId(id)) return null;
  const cookie = (await cookies()).toString();
  return fetchAsset(id, cookie);
});

const SHELL = "mx-auto w-full max-w-[var(--shell)] px-[var(--gutter)]";

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

  return (
    <WorkingCopyProvider key={asset.id} version={asset.workingCopyVersion}>
      <AssetWorkspace
        allowedApps={asset.allowedApps}
        assetId={asset.id}
        blocks={asset.blocks}
        eligibleApps={asset.eligibleApps}
        identity={{ isNsfw: asset.isNsfw, name: asset.name }}
        isDraft={isDraft}
        isOwner={asset.isOwner}
        unpublishedChanges={Boolean(asset.unpublishedChanges)}
      >
        <div className="relative isolate overflow-x-clip pb-chapter">
          <article>
            <AssetHeader
              asset={asset}
              holdsNothing={assetHoldsNothing(asset.blocks)}
              kind={kind}
              sharedDate={sharedDate}
              shellClassName={SHELL}
            />
            <AssetBlocks
              addableBlocks={asset.addableBlocks ?? []}
              assetId={asset.id}
              images={asset.media}
              isOwner={asset.isOwner}
              kind={asset.kind}
              shellClassName={SHELL}
            />
          </article>
        </div>
        <WorkspaceSurfaces
          creator={asset.creator}
          discovery={asset.discovery}
          hasOriginal={Boolean(asset.original)}
          images={asset.media}
          kind={kind.toLowerCase()}
          readiness={asset.readiness}
          sealedBlocks={asset.sealedBlocks}
          sealsPrompts={asset.linkedInstallOnly}
          unpublishedChanges={Boolean(asset.unpublishedChanges)}
          withheld={Boolean(asset.withhold)}
        />
      </AssetWorkspace>
    </WorkingCopyProvider>
  );
}
