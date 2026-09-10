/** The grounds a creator without a picture stands on, so a page of them has rhythm. */
export const PORTRAIT_GROUNDS: string[] = [
  "bg-accent-wash text-accent",
  "bg-deep text-ink",
  "bg-media text-on-media",
  "bg-rule/50 text-ink",
];

/** Spreads handles across the grounds without storing a value against any of them. */
export function portraitGround(handle: string): string {
  let hash = 0;
  for (const character of handle) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return PORTRAIT_GROUNDS[hash % PORTRAIT_GROUNDS.length];
}
