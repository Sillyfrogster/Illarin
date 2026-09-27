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
    ? `The blurb must be ${BLURB_LIMIT} characters or fewer.`
    : "";
}
