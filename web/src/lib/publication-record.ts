import {
  fetchPostArchive,
  type PostSummary,
  type PublicationApp,
  type PublicationCategory,
} from "@/lib/api/query";

export type PublicationRecord = {
  posts: PostSummary[];
  pages: number;
  category: PublicationCategory | null;
  app: PublicationApp | null;
};

export async function readPublication(
  scope: { category?: string; app?: string },
  wanted: number,
): Promise<PublicationRecord | null> {
  const posts: PostSummary[] = [];
  const first = await fetchPostArchive({ page: 1, ...scope });
  if (!first) return null;
  for (let page = 1; ; page += 1) {
    const archive =
      page === 1 ? first : await fetchPostArchive({ page, ...scope });
    if (!archive) break;
    posts.push(...archive.posts);
    if (posts.length >= wanted || page >= archive.pages) break;
  }
  return {
    posts: posts.slice(0, wanted),
    pages: first.pages,
    category: first.category ?? null,
    app: first.app ?? null,
  };
}
