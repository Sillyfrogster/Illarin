type PageElement = {
  isEmpty: boolean;
  role?: string | null;
};

const MODEL_DISCLOSURE_ROLES = new Set([
  "system_prompt",
  "post_history_instructions",
]);

function belongsInModelDisclosure(element: PageElement): boolean {
  return MODEL_DISCLOSURE_ROLES.has(element.role ?? "");
}

export function splitAssetPageContent<
  TElement extends PageElement,
  TBlock extends { hidden: boolean },
>(blocks: readonly (TBlock & { elements: TElement[] })[]) {
  const publicBlocks = blocks.map((block) => {
    const elements = block.elements.filter(
      (element) => !element.isEmpty && !belongsInModelDisclosure(element),
    );
    return { ...block, elements, empty: elements.length === 0 };
  });
  const modelContent: Array<{
    block: TBlock & { elements: TElement[] };
    element: TElement;
  }> = [];
  for (const block of blocks) {
    if (block.hidden) continue;
    for (const element of block.elements) {
      if (!element.isEmpty && belongsInModelDisclosure(element)) {
        modelContent.push({ block, element });
      }
    }
  }
  return { publicBlocks, modelContent };
}

export function rendersOnThePage(block: {
  hidden: boolean;
  empty: boolean;
}): boolean {
  return !block.hidden && !block.empty;
}

export type BlockAudience = "shown" | "hidden" | "empty" | "model";

/** Says what a reader meets where a block sits, which is not always the block */
export function blockAudience(block: {
  hidden: boolean;
  elements: readonly PageElement[];
}): BlockAudience {
  if (block.hidden) return "hidden";
  const filled = block.elements.filter((element) => !element.isEmpty);
  if (filled.length === 0) return "empty";
  if (filled.every(belongsInModelDisclosure)) return "model";
  return "shown";
}

const INVITATION_BLOCK_LIMIT = 3;

type FillableBlock = {
  title: string;
  required: boolean;
  isEmpty: boolean;
};

export function assetHoldsNothing(
  blocks: readonly Pick<FillableBlock, "isEmpty">[],
): boolean {
  return blocks.every((block) => block.isEmpty);
}

export function coreBlockTitles(
  blocks: readonly Pick<FillableBlock, "title" | "required">[],
): string[] {
  const required = blocks.filter((block) => block.required);
  const named = required.length > 0 ? required : blocks;
  return named.slice(0, INVITATION_BLOCK_LIMIT).map((block) => block.title);
}
