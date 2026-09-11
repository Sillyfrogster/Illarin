export const BLURB_LIMIT = 400;

export function blurbCharacterCount(blurb: string): number {
  return Array.from(blurb).length;
}

export function blurbLimitMessage(blurb: string): string {
  return blurbCharacterCount(blurb) > BLURB_LIMIT
    ? `The blurb must be ${BLURB_LIMIT} characters or fewer.`
    : "";
}
