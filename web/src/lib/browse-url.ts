import type { BrowseFilters, BrowseType } from "./api/query";

const TYPES = new Set<BrowseType>([
  "character",
  "lorebook",
  "preset",
  "theme",
  "pack",
  "extension",
]);

function first(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

export function readBrowseFilters(
  values: Record<string, string | string[] | undefined>,
): BrowseFilters {
  const requestedType = first(values.type) ?? first(values.kind);
  const type = TYPES.has(requestedType as BrowseType)
    ? (requestedType as BrowseType)
    : undefined;
  const q = first(values.q) || undefined;
  const platform = first(values.platform)?.trim() || undefined;
  const facets = Array.isArray(values.facet)
    ? values.facet
    : values.facet
      ? [values.facet]
      : undefined;

  return { type, q, platform, facet: facets };
}

export function buildBrowseHref(filters: BrowseFilters, basePath = "/browse") {
  const params = new URLSearchParams();
  if (filters.type) params.set("type", filters.type);
  if (filters.platform) params.set("platform", filters.platform);
  if (filters.q) params.set("q", filters.q);
  for (const facet of filters.facet ?? []) params.append("facet", facet);
  const query = params.toString();
  return query ? `${basePath}?${query}` : basePath;
}
