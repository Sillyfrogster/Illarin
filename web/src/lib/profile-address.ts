const HANDLE = /^[a-z0-9._]{3,32}$/;

const PUNCTUATION_ONLY = /^[._]+$/;

export function readProfileAddress(segment: string): string | null {
  if (!segment.startsWith("@")) return null;
  const handle = segment.slice(1).toLowerCase();
  if (!HANDLE.test(handle) || PUNCTUATION_ONLY.test(handle)) return null;
  return handle;
}

export function profilePath(handle: string): string {
  return `/@${encodeURI(handle)}`;
}
