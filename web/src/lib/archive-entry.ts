import type { PostSummary } from "@/lib/api/query";

/** What an archive has already said in its heading, which its rows need not repeat. */
export type ArchiveNarrowing = "category" | "app" | null;

/** Which facts one archive row states, once every repeat of its archive's scope is dropped. */
export type EntryFacts = {
  category: boolean;
  app: NonNullable<PostSummary["app"]> | null;
  affiliation: boolean;
};

/** The facts a row states, once everything its archive already said is dropped. */
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

/** The app a row names, which is the one its byline has not already stood for. */
function filedApp(post: PostSummary): EntryFacts["app"] {
  if (!post.app) return null;
  if (post.releaseVersion) return post.app;
  return post.app.slug === post.byline.app?.slug ? null : post.app;
}
