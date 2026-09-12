import { cookies } from "next/headers";
import { notFound, permanentRedirect } from "next/navigation";
import { fetchAsset, fetchLegacyProfile } from "@/lib/api/query";
import { assetHref, isAssetId } from "@/lib/asset-url";

export async function redirectFromLegacyAssetAddress(
  id: string,
): Promise<void> {
  if (!isAssetId(id)) notFound();
  const cookie = (await cookies()).toString();
  const asset = await fetchAsset(id, cookie);
  if (!asset) notFound();
  permanentRedirect(assetHref(asset.id, asset.name));
}

export async function redirectFromLegacyUserAddress(
  discordId: string,
): Promise<void> {
  const profile = await fetchLegacyProfile(discordId);
  if (!profile) notFound();
  permanentRedirect(`/@${profile.handle}`);
}
