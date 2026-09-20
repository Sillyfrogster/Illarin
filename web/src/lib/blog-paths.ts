/** The blog origin's front page. Every other blog path hangs off it. */
export const BLOG_HOME = "/";

export type ArchiveScope = "category";

export function postPath(slug: string): string {
  return `/${encodeURI(slug)}`;
}

export function archivePath(scope: ArchiveScope, slug: string): string {
  return `/${scope}/${encodeURI(slug)}`;
}

export function pageAddress(address: string, page: number): string {
  return page === 1 ? address : `${prefixOf(address)}/page/${page}`;
}

export function feedAddresses(address: string): { rss: string; json: string } {
  return {
    rss: `${prefixOf(address)}/feed.xml`,
    json: `${prefixOf(address)}/feed.json`,
  };
}

export const BLOG_FEEDS = feedAddresses(BLOG_HOME);

export function archivePage(paging: string[] | undefined): number | null {
  if (!paging || paging.length === 0) return 1;
  if (paging.length !== 2 || paging[0] !== "page") return null;
  if (!/^[1-9][0-9]*$/.test(paging[1])) return null;
  return Number(paging[1]);
}

function prefixOf(address: string): string {
  return address === BLOG_HOME ? "" : address;
}
