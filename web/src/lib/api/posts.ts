import type {
  Post,
  PostAction,
  PostDelivery,
  PostMedia,
  PostMediaPurpose,
  PostRevision,
  PublicationDestinationChoiceList,
} from "@/lib/api/query";
import type { PostDocument } from "@/lib/post-document";
import { ask } from "./request";

export type DraftedChanges = {
  version: number;
  categoryId: string;
  title: string;
  summary: string;
  slug: string;
  document: PostDocument;
  release?: { appId: string; version: string; address?: string } | null;
  header?: { mediaId: string; alt: string; caption?: string } | null;
  socialMediaId?: string | null;
};

export function readPosts() {
  return ask<{ posts: Post[] }>("GET", "/publication/posts");
}

export function readDeletedPosts() {
  return ask<{ posts: Post[] }>("GET", "/publication/posts?deleted=true");
}

export function startPost(draft: {
  grantId?: string;
  categoryId: string;
  title: string;
}) {
  return ask<Post>("POST", "/publication/posts", { body: draft });
}

export function readPost(id: string) {
  return ask<Post>("GET", `/publication/posts/${id}`);
}

export function saveDraftedChanges(id: string, drafted: DraftedChanges) {
  return ask<Post>("PUT", `/publication/posts/${id}`, { body: drafted });
}

export function uploadPostMedia(
  id: string,
  purpose: PostMediaPurpose,
  file: File,
) {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ purpose }));
  body.append("file", file, file.name);
  return ask<PostMedia>("POST", `/publication/posts/${id}/media`, { body });
}

export function correctPostAddress(id: string, slug: string) {
  return ask<Post>("PUT", `/publication/posts/${id}/address`, {
    body: { slug },
  });
}

export function correctPostByline(id: string, handle: string) {
  return ask<Post>("PUT", `/publication/posts/${id}/byline`, {
    body: { handle },
  });
}

export type Announcement = {
  destinationIds?: string[] | null;
  roleDestinationIds?: string[];
  note?: string;
};

export function publishPost(
  id: string,
  version: number,
  announcement: Announcement = {},
) {
  return ask<Post>("POST", `/publication/posts/${id}/publish`, {
    body: {
      version,
      ...announcement,
    },
  });
}

export function readPostDestinations(id: string) {
  return ask<PublicationDestinationChoiceList>(
    "GET",
    `/publication/posts/${id}/destinations`,
  );
}

export function readPostDeliveries(id: string) {
  return ask<{ deliveries: PostDelivery[] }>(
    "GET",
    `/publication/posts/${id}/deliveries`,
  );
}

export function readPostRevisions(id: string) {
  return ask<{ revisions: PostRevision[] }>(
    "GET",
    `/publication/posts/${id}/revisions`,
  );
}

export function keepPostVersion(id: string, version: number) {
  return ask<PostRevision>("POST", `/publication/posts/${id}/revisions`, {
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
    `/publication/posts/${id}/revisions/${revisionId}/restore`,
    { body: { version } },
  );
}

export function readPostHistory(id: string) {
  return ask<{ actions: PostAction[] }>(
    "GET",
    `/publication/posts/${id}/history`,
  );
}

export function schedulePost(
  id: string,
  version: number,
  at: string,
  announcement: Announcement = {},
) {
  return ask<Post>("POST", `/publication/posts/${id}/schedule`, {
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
  return ask<Post>("PUT", `/publication/posts/${id}/schedule`, {
    body: {
      revisionId,
      at,
    },
  });
}

export function cancelPostSchedule(id: string) {
  return ask<Post>("DELETE", `/publication/posts/${id}/schedule`);
}

export function withdrawPost(
  id: string,
  version: number,
  reason: string,
  explanation: string,
  announcement: Announcement,
) {
  return ask<Post>("POST", `/publication/posts/${id}/withdraw`, {
    body: {
      version,
      reason,
      explanation,
      ...announcement,
    },
  });
}

export function republishPost(
  id: string,
  version: number,
  revisionId: string,
  announcement: Announcement,
) {
  return ask<Post>("POST", `/publication/posts/${id}/republish`, {
    body: {
      version,
      revisionId,
      ...announcement,
    },
  });
}

export function deletePost(id: string, version: number) {
  return ask<Post>("POST", `/publication/posts/${id}/delete`, {
    body: { version },
  });
}

export function recoverPost(id: string, version: number) {
  return ask<Post>("POST", `/publication/posts/${id}/recover`, {
    body: { version },
  });
}
