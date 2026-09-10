import { ArrowLeft } from "lucide-react";
import type { Metadata } from "next";
import { cookies } from "next/headers";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import { cache } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { DefaultCover } from "@/components/media/DefaultCover";
import {
  type AssetDetail,
  fetchAsset,
  fetchAssetUpdates,
} from "@/lib/api/query";
import { assetDisplayName } from "@/lib/asset-name";
import { assetHoldsNothing } from "@/lib/asset-page-content";
import { assetHistoryHref, assetHref, isAssetId } from "@/lib/asset-url";
import { KIND_LABELS } from "@/lib/kinds";
import { protectedAppLabel } from "@/lib/protected-apps";
import { readableForMetadata } from "@/lib/site-metadata";
import { GetAsset } from "../[[...slug]]/GetAsset";
import { WithholdNotice } from "../[[...slug]]/WithholdNotice";
import { UpdateHistory } from "./UpdateHistory";

const loadAsset = cache(async (id: string): Promise<AssetDetail | null> => {
  if (!isAssetId(id)) return null;
  const cookie = (await cookies()).toString();
  return fetchAsset(id, cookie);
});

export async function generateMetadata({
  params,
}: PageProps<"/a/[id]/history">): Promise<Metadata> {
  const asset = await readableForMetadata(loadAsset((await params).id));
  if (!asset) return { title: "Not found" };
  const name = assetDisplayName(asset.name);
  return {
    title: `Update history · ${name}`,
    description: `Every version of ${name} Illarin has recorded.`,
    alternates: { canonical: assetHistoryHref(asset.id) },
    robots: asset.discovery === "unlisted" ? { index: false } : undefined,
  };
}

export default async function AssetHistoryPage({
  params,
}: PageProps<"/a/[id]/history">) {
  const { id } = await params;
  const asset = await loadAsset(id);
  if (!asset) notFound();

  const versions = await fetchAssetUpdates(id, (await cookies()).toString());
  const kind = KIND_LABELS[asset.kind].toLowerCase();
  const published = asset.lifecycle !== "draft";

  return (
    <div className={`${shellClasses} pt-6 pb-chapter`}>
      <div className="mx-auto max-w-[64rem]">
        <AssetPlate asset={asset} />

        <h1 className="mt-group font-display text-title font-medium tracking-[-0.02em] text-ink">
          Update history
        </h1>
        <p className="mt-3 max-w-[62ch] font-prose text-lede text-mute">
          Every version of this {kind} Illarin has recorded, and what changed
          between them. The creator’s own changelog, where they wrote one, stays
          on the {kind}’s page.
        </p>

        {asset.withhold ? <WithholdNotice withhold={asset.withhold} /> : null}

        <UpdateHistory
          assetId={asset.id}
          download={
            published ? (
              <>
                <GetAsset
                  assetId={asset.id}
                  blocks={asset.blocks}
                  downloads={asset.downloads}
                  holdsNothing={assetHoldsNothing(asset.blocks)}
                  images={asset.media}
                  isOwner={asset.isOwner}
                  kindLabel={kind}
                  linkedInstallOnly={asset.linkedInstallOnly}
                  original={asset.original}
                />
                {asset.linkedInstallOnly ? (
                  <p className="max-w-[60ch] text-meta text-mute">
                    This {kind} installs only through a linked app, so Illarin
                    writes no file of it. Allowed apps:{" "}
                    {asset.allowedApps.map(protectedAppLabel).join(", ")}.
                  </p>
                ) : null}
              </>
            ) : null
          }
          kind={kind}
          versions={versions}
        />
      </div>
    </div>
  );
}

function AssetPlate({ asset }: { asset: AssetDetail }) {
  const cover = asset.media.find((image) => image.isCover) ?? asset.media[0];

  return (
    <Link
      className="group inline-flex max-w-full items-center gap-4 outline-offset-3"
      href={assetHref(asset.id, asset.name)}
    >
      <span className="relative size-14 shrink-0 overflow-hidden rounded-control bg-deep">
        {cover ? (
          <Image
            alt=""
            className="size-full object-cover"
            height={112}
            src={cover.thumbUrl}
            unoptimized
            width={112}
          />
        ) : (
          <DefaultCover compact kind={asset.kind} />
        )}
      </span>
      <span className="min-w-0">
        <span className="flex items-center gap-2 text-ui font-medium text-ink">
          <ArrowLeft
            aria-hidden="true"
            className="size-4 text-mute transition-transform duration-200 group-hover:-translate-x-1 motion-reduce:transition-none"
          />
          <span className="truncate underline decoration-transparent underline-offset-4 transition-colors group-hover:decoration-accent motion-reduce:transition-none">
            {assetDisplayName(asset.name)}
          </span>
        </span>
        <span className="mt-0.5 block text-meta text-mute">
          {KIND_LABELS[asset.kind]} by {asset.creator}
        </span>
      </span>
    </Link>
  );
}
