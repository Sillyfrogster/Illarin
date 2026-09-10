import type { PostSummary } from "@/lib/api/query";

export type ArchiveNarrowing = "category" | "app" | null;

export type EntryFacts = {
  category: boolean;
  app: NonNullable<PostSummary["app"]> | null;
  affiliation: boolean;
};

export function entryFacts(
  post: PostSummary,
  narrowed: ArchiveNarrowing,
): EntryFacts {
  const app = narrowed === "app" ? null : filedApp(post);
  return {
    category: narrowed !== "category",
    app,
    affiliation:
      narrowed !== "app" &&
      (app === null || app.slug !== post.byline.app?.slug),
  };
}

function filedApp(post: PostSummary): EntryFacts["app"] {
  if (!post.app) return null;
  if (post.releaseVersion) return post.app;
  return post.app.slug === post.byline.app?.slug ? null : post.app;
}
