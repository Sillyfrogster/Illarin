import type { Post, PostMedia, PostMediaPurpose } from "@/lib/api/query";
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

export function publishPost(id: string) {
  return ask<Post>(
    `/publication/posts/${id}/publish`,
    { method: "POST" },
    (response) => response.json() as Promise<Post>,
  );
}
