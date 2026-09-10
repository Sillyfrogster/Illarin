const SLUG_LIMIT = 60;

const ASSET_ID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const RESERVED = new Set(["history"]);

export function isAssetId(segment: string): boolean {
  return ASSET_ID.test(segment);
}

export function assetSlug(name: string): string {
  const normalized = name
    .normalize("NFKD")
    .replace(/\p{M}+/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");

  if (RESERVED.has(normalized)) return "";
  if (normalized.length <= SLUG_LIMIT) return normalized;

  const capped = normalized.slice(0, SLUG_LIMIT);
  const boundary = capped.lastIndexOf("-");
  return boundary > 0 ? capped.slice(0, boundary) : capped;
}

export function assetHref(id: string, name: string): string {
  const slug = assetSlug(name);
  return slug ? `/a/${id}/${slug}` : `/a/${id}`;
}

export function assetHistoryHref(id: string): string {
  return `/a/${id}/history`;
}

export function assetRedirect(
  visited: { id: string; slug?: string[] },
  asset: { id: string; name: string },
): string | null {
  const here = visited.slug?.length
    ? `/a/${visited.id}/${visited.slug.join("/")}`
    : `/a/${visited.id}`;
  const canonical = assetHref(asset.id, asset.name);
  return here === canonical ? null : canonical;
}
