export const BLURB_LIMIT = 400;

type DetailsFields = {
  blurb: string;
  isNsfw: boolean | null;
  name: string;
  tags: string[];
};

export function detailsHasChanged(
  draft: DetailsFields,
  saved: DetailsFields,
): boolean {
  return (
    draft.name !== saved.name ||
    draft.blurb !== saved.blurb ||
    draft.isNsfw !== saved.isNsfw ||
    draft.tags.length !== saved.tags.length ||
    draft.tags.some((tag, index) => tag !== saved.tags[index])
  );
}

export function blurbCharacterCount(blurb: string): number {
  return Array.from(blurb).length;
}

export function blurbLimitMessage(blurb: string): string {
  return blurbCharacterCount(blurb) > BLURB_LIMIT
    ? `Keep the blurb to ${BLURB_LIMIT} characters.`
    : "";
}

export const TAG_LIMIT = 32;

/** tagTrouble says why a tag cannot be added, or nothing when it can. */
export function tagTrouble(tags: readonly string[], tag: string): string {
  if (Array.from(tag).length > 64)
    return "Use 1 to 64 characters for each tag.";
  if (tags.includes(tag)) return "That tag is already here.";
  if (tags.length >= TAG_LIMIT) return `Use up to ${TAG_LIMIT} tags.`;
  return "";
}
