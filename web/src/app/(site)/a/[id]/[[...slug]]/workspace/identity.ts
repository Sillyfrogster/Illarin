export const BLURB_LIMIT = 400;

type IdentityFields = {
  blurb: string;
  isNsfw: boolean | null;
  name: string;
};

export function identityHasChanged(
  draft: IdentityFields,
  saved: IdentityFields,
): boolean {
  return (
    draft.name !== saved.name ||
    draft.blurb !== saved.blurb ||
    draft.isNsfw !== saved.isNsfw
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
