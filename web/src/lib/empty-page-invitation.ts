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
    const fill = `Add content to ${named}.`;
    return canAdd
      ? `${fill} Select the block to edit it, or choose Add block for more content.`
      : `${fill} Select the block to edit it.`;
  }

  return canAdd
    ? `Choose Add block to add content to this ${kindLabel}.`
    : `No editable blocks are available for this ${kindLabel}.`;
}

function namedInSentence(titles: readonly string[]): string {
  if (titles.length <= 1) return titles[0] ?? "";
  if (titles.length === 2) return titles.join(" and ");
  return `${titles.slice(0, -1).join(", ")} and ${titles.at(-1)}`;
}
