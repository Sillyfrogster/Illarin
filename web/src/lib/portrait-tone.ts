export const PORTRAIT_GROUNDS: string[] = [
  "bg-accent-wash text-accent",
  "bg-deep text-ink",
  "bg-media text-on-media",
  "bg-rule/50 text-ink",
];

export function portraitGround(handle: string): string {
  let hash = 0;
  for (const character of handle) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % 100000;
  }
  return PORTRAIT_GROUNDS[hash % PORTRAIT_GROUNDS.length];
}
