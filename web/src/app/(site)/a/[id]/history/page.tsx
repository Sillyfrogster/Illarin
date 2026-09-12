import { cookies } from "next/headers";
import { notFound, permanentRedirect } from "next/navigation";
import { fetchAsset } from "@/lib/api/query";
import { assetHistoryHref, isAssetId } from "@/lib/asset-url";

/** Sends the old history address to the asset page, which keeps any version fragment. */
export default async function AssetHistoryPage({
  params,
}: PageProps<"/a/[id]/history">) {
  const { id } = await params;
  if (!isAssetId(id)) notFound();
  const asset = await fetchAsset(id, (await cookies()).toString());
  if (!asset) notFound();
  permanentRedirect(assetHistoryHref(asset.id, asset.name));
}
