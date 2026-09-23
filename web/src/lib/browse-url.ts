import type { BrowseFilters } from "./api/query";
import { isWorkType } from "./work-types";

function first(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

export function readBrowseFilters(
  values: Record<string, string | string[] | undefined>,
): BrowseFilters {
  const requestedType = first(values.type) ?? first(values.kind);
  const type = isWorkType(requestedType) ? requestedType : undefined;
  const q = first(values.q) || undefined;
  const facets = Array.isArray(values.facet)
    ? values.facet
    : values.facet
      ? [values.facet]
      : undefined;

  return { type, q, facet: facets };
}

export function buildBrowseHref(filters: BrowseFilters, basePath = "/browse") {
  const params = new URLSearchParams();
  if (filters.type) params.set("type", filters.type);
  if (filters.q) params.set("q", filters.q);
  for (const facet of filters.facet ?? []) params.append("facet", facet);
  const query = params.toString();
  return query ? `${basePath}?${query}` : basePath;
}

/** Sets one filter to a value, or clears it with null, keeping the rest. */
export function chooseFilter(
  filters: BrowseFilters,
  key: string,
  value: string | null,
): BrowseFilters {
  const kept = (filters.facet ?? []).filter(
    (one) => !one.startsWith(`${key}=`),
  );
  const facet = value === null ? kept : [...kept, `${key}=${value}`];
  return { ...filters, facet: facet.length ? facet : undefined };
}
