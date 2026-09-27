import type { BrowseFilters, BrowseSort } from "./api/query";
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

  const sort = first(values.sort);

  return {
    type,
    q,
    facet: facets,
    sort: isActivitySort(sort) ? sort : undefined,
  };
}

function isActivitySort(value: string | undefined): value is BrowseSort {
  return value === "downloads" || value === "views";
}

export function buildBrowseHref(filters: BrowseFilters, basePath = "/browse") {
  const params = new URLSearchParams();
  if (filters.type) params.set("type", filters.type);
  if (filters.q) params.set("q", filters.q);
  for (const facet of filters.facet ?? []) params.append("facet", facet);
  if (filters.sort && filters.sort !== "recent")
    params.set("sort", filters.sort);
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
