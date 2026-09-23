import type {
  BlogCategory,
  BlogWorkspace,
  WriterList,
  WriterResponse,
} from "@/lib/api/query";
import { ask } from "./request";

export function readCategories() {
  return ask<{ categories: BlogCategory[] }>("GET", "/blog/categories");
}

export function updateCategory(
  id: string,
  change: { label?: string; retired?: boolean },
) {
  return ask<BlogCategory>("PATCH", `/blog/categories/${id}`, {
    body: change,
  });
}

export function orderCategories(categoryIds: string[]) {
  return ask<{ categories: BlogCategory[] }>("PUT", "/blog/categories", {
    body: { categoryIds },
  });
}

export function readWriters() {
  return ask<WriterList>("GET", "/blog/writers");
}

export function switchWriterOn(handle: string) {
  return ask<WriterResponse>("POST", "/blog/writers", { body: { handle } });
}

export function switchWriterOff(accountId: string) {
  return ask<void>("DELETE", `/blog/writers/${accountId}`);
}

export function readWorkspace() {
  return ask<BlogWorkspace>("GET", "/blog/workspace");
}
