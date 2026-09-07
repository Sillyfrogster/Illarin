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
import { ask, json } from "./distinctions";

export type WorkingCopy = {
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
  return json<{ posts: Post[] }>("/publication/posts", "GET");
}

export function readDeletedPosts() {
  return json<{ posts: Post[] }>("/publication/posts?deleted=true", "GET");
}

export function startPost(draft: {
  grantId?: string;
  categoryId: string;
  title: string;
}) {
  return json<Post>("/publication/posts", "POST", draft);
}

export function readPost(id: string) {
  return json<Post>(`/publication/posts/${id}`, "GET");
}

export function saveWorkingCopy(id: string, working: WorkingCopy) {
  return json<Post>(`/publication/posts/${id}`, "PUT", working);
}

export function uploadPostMedia(
  id: string,
  purpose: PostMediaPurpose,
  file: File,
) {
  const body = new FormData();
  body.append("metadata", JSON.stringify({ purpose }));
  body.append("file", file, file.name);
  return ask<PostMedia>(
    `/publication/posts/${id}/media`,
    { method: "POST", body },
    (response) => response.json() as Promise<PostMedia>,
  );
}

export function correctPostAddress(id: string, slug: string) {
  return json<Post>(`/publication/posts/${id}/address`, "PUT", { slug });
}

export function correctPostByline(id: string, handle: string) {
  return json<Post>(`/publication/posts/${id}/byline`, "PUT", { handle });
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
  return json<Post>(`/publication/posts/${id}/publish`, "POST", {
    version,
    ...announcement,
  });
}

export function readPostDestinations(id: string) {
  return json<PublicationDestinationChoiceList>(
    `/publication/posts/${id}/destinations`,
    "GET",
  );
}

export function readPostDeliveries(id: string) {
  return json<{ deliveries: PostDelivery[] }>(
    `/publication/posts/${id}/deliveries`,
    "GET",
  );
}

export function readPostRevisions(id: string) {
  return json<{ revisions: PostRevision[] }>(
    `/publication/posts/${id}/revisions`,
    "GET",
  );
}

export function keepPostVersion(id: string, version: number) {
  return json<PostRevision>(`/publication/posts/${id}/revisions`, "POST", {
    version,
  });
}

export function restorePostRevision(
  id: string,
  revisionId: string,
  version: number,
) {
  return json<Post>(
    `/publication/posts/${id}/revisions/${revisionId}/restore`,
    "POST",
    { version },
  );
}

export function readPostHistory(id: string) {
  return json<{ actions: PostAction[] }>(
    `/publication/posts/${id}/history`,
    "GET",
  );
}

export function schedulePost(
  id: string,
  version: number,
  at: string,
  announcement: Announcement = {},
) {
  return json<Post>(`/publication/posts/${id}/schedule`, "POST", {
    version,
    at,
    ...announcement,
  });
}

export function replacePostSchedule(
  id: string,
  revisionId: string,
  at: string,
) {
  return json<Post>(`/publication/posts/${id}/schedule`, "PUT", {
    revisionId,
    at,
  });
}

export function cancelPostSchedule(id: string) {
  return json<Post>(`/publication/posts/${id}/schedule`, "DELETE");
}

export function withdrawPost(
  id: string,
  version: number,
  reason: string,
  explanation: string,
  announcement: Announcement,
) {
  return json<Post>(`/publication/posts/${id}/withdraw`, "POST", {
    version,
    reason,
    explanation,
    ...announcement,
  });
}

export function republishPost(
  id: string,
  version: number,
  revisionId: string,
  announcement: Announcement,
) {
  return json<Post>(`/publication/posts/${id}/republish`, "POST", {
    version,
    revisionId,
    ...announcement,
  });
}

export function deletePost(id: string, version: number) {
  return json<Post>(`/publication/posts/${id}/delete`, "POST", { version });
}

export function recoverPost(id: string, version: number) {
  return json<Post>(`/publication/posts/${id}/recover`, "POST", { version });
}
