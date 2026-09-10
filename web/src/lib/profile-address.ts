import { siteUrl } from "./site-metadata";

const HANDLE = /^[a-z0-9._]{3,32}$/;

const PUNCTUATION_ONLY = /^[._]+$/;

export type ProfileAddress = {
  form: "canonical" | "legacy";
  handle: string;
};

export function readProfileAddress(segment: string): ProfileAddress | null {
  const isCanonical = segment.startsWith("@");
  const handle = (isCanonical ? segment.slice(1) : segment).toLowerCase();
  if (!HANDLE.test(handle) || PUNCTUATION_ONLY.test(handle)) return null;
  return { form: isCanonical ? "canonical" : "legacy", handle };
}

export function profileAddress(handle: string): string {
  return new URL(`/@${encodeURI(handle)}`, siteUrl).href;
}
