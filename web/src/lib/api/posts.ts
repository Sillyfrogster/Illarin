import type {
  Post,
  PostAction,
  PostMedia,
  PostMediaPurpose,
  PostRevision,
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

export function publishPost(id: string, version: number) {
  return json<Post>(`/publication/posts/${id}/publish`, "POST", { version });
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
