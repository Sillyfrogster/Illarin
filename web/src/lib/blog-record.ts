import {
  type BlogCategory,
  fetchPostArchive,
  type PostSummary,
} from "@/lib/api/query";

export type BlogRecord = {
  posts: PostSummary[];
  pages: number;
  category: BlogCategory | null;
};

export async function readBlog(
  scope: { category?: string },
  wanted: number,
): Promise<BlogRecord | null> {
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
  };
}
