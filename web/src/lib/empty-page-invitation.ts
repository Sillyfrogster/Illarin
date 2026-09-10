export function emptyPageInvitation({
  coreBlocks,
  canAdd,
  kindLabel,
}: {
  coreBlocks: readonly string[];
  canAdd: boolean;
  kindLabel: string;
}): string {
  const named = namedInSentence(coreBlocks);

  if (named) {
    const fill = `Fill in ${named} to give the page something to show.`;
    return canAdd
      ? `${fill} Edit block opens it, and Add block brings in anything else a ${kindLabel} can hold.`
      : `${fill} Edit block opens it.`;
  }

  return canAdd
    ? `Add block brings in the first of what a ${kindLabel} can hold.`
    : `Illarin has no blocks for a ${kindLabel} yet. The file you uploaded is kept whole, and every download carries it.`;
}

function namedInSentence(titles: readonly string[]): string {
  if (titles.length <= 1) return titles[0] ?? "";
  if (titles.length === 2) return titles.join(" and ");
  return `${titles.slice(0, -1).join(", ")} and ${titles.at(-1)}`;
}
