export const BLOG_HOME = "/blog";

export type ArchiveScope = "category";

export function postPath(slug: string): string {
  return `${BLOG_HOME}/${encodeURI(slug)}`;
}

export function archivePath(scope: ArchiveScope, slug: string): string {
  return `${BLOG_HOME}/${scope}/${encodeURI(slug)}`;
}

export function pageAddress(address: string, page: number): string {
  return page === 1 ? address : `${address}/page/${page}`;
}

export function feedAddresses(address: string): { rss: string; json: string } {
  return {
    rss: `${address}/feed.xml`,
    json: `${address}/feed.json`,
  };
}

export const BLOG_FEEDS = feedAddresses(BLOG_HOME);

export function archivePage(paging: string[] | undefined): number | null {
  if (!paging || paging.length === 0) return 1;
  if (paging.length !== 2 || paging[0] !== "page") return null;
  if (!/^[1-9][0-9]*$/.test(paging[1])) return null;
  return Number(paging[1]);
}
