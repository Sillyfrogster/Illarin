/** The creator's writing is the page's primary text, so it is set in ink. */
export const PROSE = "text-prose text-ink/92 [&_p]:text-prose";

/** Reading matter sits on the page's own field at the reading measure. */
export const PASSAGE = "flex list-none flex-col gap-7";

export const PASSAGE_NAME =
  "mb-2 font-ui text-meta font-semibold tracking-[0.01em] text-ink";

/** An element is named at the size between a block and its items. */
export const ELEMENT_NAME =
  "font-display text-article leading-snug font-medium text-ink [overflow-wrap:anywhere]";

export const ITEM_NAME =
  "font-ui text-ui font-medium text-ink [overflow-wrap:anywhere]";

export const ITEM_META = "font-ui text-meta text-mute [overflow-wrap:anywhere]";

export const ITEM_BODY = "text-ui text-ink/85 [&_p]:!text-ui [&_p]:!leading-7";

export const ITEM_VALUE =
  "font-ui text-ui font-medium text-ink tabular-nums [overflow-wrap:anywhere]";

/** A mark that names a state rather than repeating a sentence on every row. */
export const TAG =
  "inline-flex min-h-6 shrink-0 items-center gap-1.5 rounded-control px-2 font-ui text-label font-medium";

export const OFF = "opacity-55";

export const CODE =
  "overflow-x-auto rounded-control bg-deep p-4 font-mono text-meta leading-7 whitespace-pre-wrap [overflow-wrap:anywhere]";
