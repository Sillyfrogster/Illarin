/** The creator's writing is the page's primary text, so it is set in ink. */
export const PROSE = "text-prose text-ink/92 [&_p]:text-prose";

/** Reading matter sits on the page's own field at the reading measure. */
export const PASSAGE = "flex list-none flex-col gap-7";

export const PASSAGE_NAME =
  "mb-2 font-ui text-meta font-semibold tracking-[0.01em] text-ink";

/** Illarin names a field quietly, so the creator's own headings stay the loudest. */
export const ELEMENT_NAME =
  "flex min-w-0 flex-1 basis-60 items-center gap-3 font-ui text-meta font-semibold tracking-[0.09em] text-mute uppercase [overflow-wrap:anywhere]";

/** A hairline that marks the label as Illarin's rather than the creator's. */
export const ELEMENT_RULE = "h-px min-w-4 flex-1 bg-rule/70";

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
