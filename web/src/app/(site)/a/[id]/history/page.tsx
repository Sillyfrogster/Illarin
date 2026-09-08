import { ArrowLeft } from "lucide-react";
import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import { notFound } from "next/navigation";
import { cache } from "react";
import { Shell } from "@/components/layout/Shell";
import {
  type AssetDetail,
  fetchAsset,
  fetchAssetUpdates,
} from "@/lib/api/query";
import { assetDisplayName } from "@/lib/asset-name";
import { assetHistoryHref, assetHref, isAssetId } from "@/lib/asset-url";
import { KIND_LABELS } from "@/lib/kinds";
import styles from "./HistoryPage.module.css";
import { UpdateHistory } from "./UpdateHistory";

const loadAsset = cache(async (id: string): Promise<AssetDetail | null> => {
  if (!isAssetId(id)) return null;
  const cookie = (await cookies()).toString();
  return fetchAsset(id, cookie);
});

export async function generateMetadata({
  params,
}: PageProps<"/a/[id]/history">): Promise<Metadata> {
  const asset = await loadAsset((await params).id);
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

  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <Link href={assetHref(asset.id, asset.name)} className={styles.back}>
          <ArrowLeft size={15} aria-hidden="true" />
          Back to {assetDisplayName(asset.name)}
        </Link>

        <header className={styles.heading}>
          <h1>Update history</h1>
          <p>
            Every version of this {kind} Illarin has recorded, and what changed
            between them. The creator's own changelog, where they wrote one,
            stays on the {kind}'s page.
          </p>
        </header>

        <UpdateHistory assetId={asset.id} kind={kind} versions={versions} />
      </Shell>
    </section>
  );
}
