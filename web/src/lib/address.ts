/** An address on one origin whose path and query come from the caller, so a path can only ever name a page there. */
export function addressOn(origin: string, path: string, search = ""): string {
  const address = new URL(origin);
  address.pathname = path;
  address.search = search;
  return address.href;
}
