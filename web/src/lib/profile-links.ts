export type LinkBrand =
  | "bluesky"
  | "discord"
  | "github"
  | "itch"
  | "ko-fi"
  | "patreon"
  | "pixiv"
  | "reddit"
  | "telegram"
  | "tumblr"
  | "twitch"
  | "x"
  | "youtube";

const BRANDS: [RegExp, LinkBrand][] = [
  [/(^|\.)bsky\.app$/, "bluesky"],
  [/(^|\.)discord\.(gg|com)$/, "discord"],
  [/(^|\.)github\.com$/, "github"],
  [/(^|\.)itch\.io$/, "itch"],
  [/(^|\.)ko-fi\.com$/, "ko-fi"],
  [/(^|\.)patreon\.com$/, "patreon"],
  [/(^|\.)pixiv\.net$/, "pixiv"],
  [/(^|\.)reddit\.com$/, "reddit"],
  [/(^|\.)(t\.me|telegram\.org)$/, "telegram"],
  [/(^|\.)tumblr\.com$/, "tumblr"],
  [/(^|\.)twitch\.tv$/, "twitch"],
  [/(^|\.)(x|twitter)\.com$/, "x"],
  [/(^|\.)(youtube\.com|youtu\.be)$/, "youtube"],
];

/** Names the site a profile link points at when it is one a roleplay Discord would recognize. */
export function linkBrand(address: string): LinkBrand | null {
  const host = linkHost(address);
  if (!host) return null;
  return BRANDS.find(([pattern]) => pattern.test(host))?.[1] ?? null;
}

/** The host a link points at, without www, or empty when the address is not one. */
export function linkHost(address: string): string {
  try {
    return new URL(address).hostname.replace(/^www\./, "").toLowerCase();
  } catch {
    return "";
  }
}

/** The host a link points at, plus its path when that says more */
export function linkText(address: string): string {
  try {
    const url = new URL(address);
    const host = url.hostname.replace(/^www\./, "");
    const path = url.pathname.replace(/\/$/, "");
    const shown = path && path !== "/" ? `${host}${path}` : host;
    return shown.length > 40 ? `${shown.slice(0, 39)}…` : shown;
  } catch {
    return address;
  }
}
