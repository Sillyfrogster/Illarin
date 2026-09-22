import type {
  Post,
  PostAction,
  PostMedia,
  PostMediaPurpose,
  PostRevision,
} from "@/lib/api/query";
import type { PostBody } from "@/lib/post-body";
import { ask } from "./request";

export type DraftedChanges = {
  version: number;
  categoryId: string;
  title: string;
  summary: string;
  slug: string;
  body: PostBody;
  release?: { appId: string; version: string; address?: string } | null;
  header?: { mediaId: string; alt: string; caption?: string } | null;
  linkCardMediaId?: string | null;
};

export function readPosts() {
  return ask<{ posts: Post[] }>("GET", "/blog/posts");
}

export function readDeletedPosts() {
  return ask<{ posts: Post[] }>("GET", "/blog/posts?deleted=true");
}

export function startPost(draft: { categoryId: string; title: string }) {
  return ask<Post>("POST", "/blog/posts", { body: draft });
}

export function readPost(id: string) {
  return ask<Post>("GET", `/blog/posts/${id}`);
}

export function saveDraftedChanges(id: string, drafted: DraftedChanges) {
  return ask<Post>("PUT", `/blog/posts/${id}`, { body: drafted });
}

export function uploadPostMedia(
  id: string,
  purpose: PostMediaPurpose,
  file: File,
) {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ purpose }));
  body.append("file", file, file.name);
  return ask<PostMedia>("POST", `/blog/posts/${id}/media`, { body });
}

export function correctPostAddress(id: string, slug: string) {
  return ask<Post>("PUT", `/blog/posts/${id}/address`, {
    body: { slug },
  });
}

export function correctPostByline(id: string, handle: string) {
  return ask<Post>("PUT", `/blog/posts/${id}/byline`, {
    body: { handle },
  });
}

/** Announcement says whether a post's first publication goes to the blog's Discord channel. */
export type Announcement = { discord?: boolean };

export function publishPost(
  id: string,
  version: number,
  announcement: Announcement = {},
) {
  return ask<Post>("POST", `/blog/posts/${id}/publish`, {
    body: {
      version,
      ...announcement,
    },
  });
}

export function readPostRevisions(id: string) {
  return ask<{ revisions: PostRevision[] }>(
    "GET",
    `/blog/posts/${id}/revisions`,
  );
}

export function keepPostVersion(id: string, version: number) {
  return ask<PostRevision>("POST", `/blog/posts/${id}/revisions`, {
    body: {
      version,
    },
  });
}

export function restorePostRevision(
  id: string,
  revisionId: string,
  version: number,
) {
  return ask<Post>(
    "POST",
    `/blog/posts/${id}/revisions/${revisionId}/restore`,
    { body: { version } },
  );
}

export function readPostHistory(id: string) {
  return ask<{ actions: PostAction[] }>("GET", `/blog/posts/${id}/history`);
}

export function schedulePost(
  id: string,
  version: number,
  at: string,
  announcement: Announcement = {},
) {
  return ask<Post>("POST", `/blog/posts/${id}/schedule`, {
    body: {
      version,
      at,
      ...announcement,
    },
  });
}

export function replacePostSchedule(
  id: string,
  revisionId: string,
  at: string,
) {
  return ask<Post>("PUT", `/blog/posts/${id}/schedule`, {
    body: {
      revisionId,
      at,
    },
  });
}

export function cancelPostSchedule(id: string) {
  return ask<Post>("DELETE", `/blog/posts/${id}/schedule`);
}

export function unpublishPost(
  id: string,
  version: number,
  reason: string,
  explanation: string,
) {
  return ask<Post>("POST", `/blog/posts/${id}/unpublish`, {
    body: {
      version,
      reason,
      explanation,
    },
  });
}

export function republishPost(id: string, version: number, revisionId: string) {
  return ask<Post>("POST", `/blog/posts/${id}/republish`, {
    body: {
      version,
      revisionId,
    },
  });
}

export function deletePost(id: string, version: number) {
  return ask<Post>("POST", `/blog/posts/${id}/delete`, {
    body: { version },
  });
}

export function recoverPost(id: string, version: number) {
  return ask<Post>("POST", `/blog/posts/${id}/recover`, {
    body: { version },
  });
}
