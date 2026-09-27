const COMPACT = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

/** Says a work's lifetime views and downloads compactly, zero included */
export function workCounts(views: number, downloads: number): string {
  return `${counted(views, "view")} · ${counted(downloads, "download")}`;
}

function counted(value: number, one: string) {
  return `${COMPACT.format(value)} ${value === 1 ? one : `${one}s`}`;
}
