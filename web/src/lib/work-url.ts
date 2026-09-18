const SLUG_LIMIT = 60;

const WORK_ID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const RESERVED = new Set(["history"]);

export function isWorkId(segment: string): boolean {
  return WORK_ID.test(segment);
}

export function workSlug(name: string): string {
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

export function workHref(id: string, name: string): string {
  const slug = workSlug(name);
  return slug ? `/a/${id}/${slug}` : `/a/${id}`;
}

/** Opens the work page with its update history showing, at one version when given. */
export function workHistoryHref(
  id: string,
  name: string,
  version?: number,
): string {
  const anchor = version === undefined ? "" : `#version-${version}`;
  return `${workHref(id, name)}?history${anchor}`;
}

export function workRedirect(
  visited: { id: string; slug?: string[] },
  work: { id: string; name: string },
): string | null {
  const here = visited.slug?.length
    ? `/a/${visited.id}/${visited.slug.join("/")}`
    : `/a/${visited.id}`;
  const canonical = workHref(work.id, work.name);
  return here === canonical ? null : canonical;
}
