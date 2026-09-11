export function isSafeAddress(href: string): boolean {
  const address = href.trim();
  for (const letter of address) {
    const code = letter.codePointAt(0) ?? 0;
    if (code <= 0x20 || code === 0x7f) return false;
  }
  return (
    (address.startsWith("https://") && address.length > "https://".length) ||
    (address.startsWith("mailto:") && address.length > "mailto:".length)
  );
}

/** Whether a link leaves Illarin, given the main site's own address. */
export function leavesIllarin(href: string, site: string): boolean {
  if (!href.startsWith("https://")) return false;
  try {
    const going = new URL(href).hostname.toLowerCase();
    const here = new URL(site).hostname.toLowerCase();
    return going !== here && !going.endsWith(`.${here}`);
  } catch {
    return true;
  }
}

export function normalizedSlug(candidate: string): string {
  return candidate
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/, "");
}
