import type { Post } from "@/lib/api/query";
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

export function publishPost(id: string) {
  return ask<Post>(
    `/publication/posts/${id}/publish`,
    { method: "POST" },
    (response) => response.json() as Promise<Post>,
  );
}
