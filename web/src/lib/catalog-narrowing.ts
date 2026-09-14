import type { BrowseFilters, BrowseKind, BrowsePage } from "./api/query";

export type Narrowing = {
  id: string;
  group: string;
  label: string;
  without: BrowseFilters;
};

const KIND_PLURALS: Record<BrowseKind, string> = {
  character: "Characters",
  lorebook: "Lorebooks",
  preset: "Presets",
  theme: "Themes",
  pack: "Packs",
  extension: "Extensions",
};

function settled(filters: BrowseFilters): BrowseFilters {
  const settled: BrowseFilters = {};
  if (filters.kind) settled.kind = filters.kind;
  if (filters.platform) settled.platform = filters.platform;
  if (filters.q) settled.q = filters.q;
  if (filters.facet?.length) settled.facet = filters.facet;
  return settled;
}

function facetLabel(encoded: string, overview: BrowsePage | null): string {
  const divide = encoded.indexOf("=");
  const key = divide === -1 ? encoded : encoded.slice(0, divide);
  const value = divide === -1 ? "" : encoded.slice(divide + 1);
  const facet = overview?.facets.find((one) => one.key === key);
  const option = facet?.options.find((one) => one.value === value);
  return option?.label ?? value ?? encoded;
}

export function narrowingsInForce(
  filters: BrowseFilters,
  overview: BrowsePage | null,
): Narrowing[] {
  const narrowings: Narrowing[] = [];

  if (filters.kind) {
    narrowings.push({
      id: `kind:${filters.kind}`,
      group: "Kind",
      label: KIND_PLURALS[filters.kind],
      without: settled({ ...filters, kind: undefined, facet: undefined }),
    });
  }

  if (filters.platform) {
    const app = overview?.platforms.find(
      (one) => one.value === filters.platform,
    );
    narrowings.push({
      id: `platform:${filters.platform}`,
      group: "Works with",
      label: app?.label ?? filters.platform,
      without: settled({ ...filters, platform: undefined }),
    });
  }

  for (const encoded of filters.facet ?? []) {
    narrowings.push({
      id: `facet:${encoded}`,
      group: "Includes",
      label: facetLabel(encoded, overview),
      without: settled({
        ...filters,
        facet: (filters.facet ?? []).filter((one) => one !== encoded),
      }),
    });
  }

  if (filters.q) {
    narrowings.push({
      id: `q:${filters.q}`,
      group: "Search",
      label: filters.q,
      without: settled({ ...filters, q: undefined }),
    });
  }

  return narrowings;
}
