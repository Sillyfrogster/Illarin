import type {
  AssetBlock,
  AssetElement,
  SaveAssetBlockRequest,
} from "@/lib/api/query";
import { writesInPlace as writtenInPlace } from "@/lib/page-arrangement";
import type { AllowedApp } from "../SealedPolicy";

/** The element types the page itself can write. Everything else keeps its own editor. */
export function writesInPlace(element: AssetElement): boolean {
  return writtenInPlace(element.type);
}

export function blockSaveRequest(
  block: AssetBlock,
  changes: {
    title?: string | null;
    layout?: AssetBlock["layout"];
    width?: AssetBlock["width"];
    elements?: AssetElement[];
    allowedApps?: AllowedApp[];
    exposeProtected?: boolean;
  } = {},
): SaveAssetBlockRequest {
  return {
    title:
      changes.title !== undefined
        ? changes.title
        : block.titleIsDefault
          ? null
          : block.title,
    layout: changes.layout ?? block.layout,
    width: changes.width ?? block.width,
    allowedApps: changes.allowedApps,
    exposeProtected: changes.exposeProtected,
    elements: (changes.elements ?? block.elements).map((element) => ({
      id: element.id,
      type: element.type,
      role: element.role,
      slot: element.slot,
      display: element.display,
      itemSize: element.itemSize,
      content: element.content,
    })),
  };
}

export function replaceBlock(
  blocks: AssetBlock[],
  saved: AssetBlock,
): AssetBlock[] {
  return blocks.map((block) => (block.id === saved.id ? saved : block));
}

export function replaceElement(
  block: AssetBlock,
  element: AssetElement,
): AssetBlock {
  return {
    ...block,
    elements: block.elements.map((item) =>
      item.id === element.id ? element : item,
    ),
  };
}

/** Which blocks a creator has written since the last save, compared by content alone. */
export function changedBlockIds(
  draft: AssetBlock[],
  saved: AssetBlock[],
): string[] {
  const before = new Map(saved.map((block) => [block.id, block]));
  return draft
    .filter((block) => {
      const original = before.get(block.id);
      return original ? !sameContent(original, block) : true;
    })
    .map((block) => block.id);
}

function sameContent(left: AssetBlock, right: AssetBlock): boolean {
  return (
    left.title === right.title &&
    left.layout === right.layout &&
    left.width === right.width &&
    JSON.stringify(left.elements.map(contentOf)) ===
      JSON.stringify(right.elements.map(contentOf))
  );
}

function contentOf(element: AssetElement) {
  return {
    id: element.id,
    display: element.display,
    itemSize: element.itemSize,
    content: element.content,
  };
}

export function isEmptyContent(element: AssetElement): boolean {
  const content = element.content as Record<string, unknown>;
  if ("text" in content) return String(content.text ?? "").trim() === "";
  if ("texts" in content) {
    return (content.texts as { text: string }[]).every(
      (item) => item.text.trim() === "",
    );
  }
  if ("turns" in content) {
    return (content.turns as { text: string }[]).every(
      (turn) => turn.text.trim() === "",
    );
  }
  if ("fields" in content) {
    return (content.fields as { value: string }[]).every(
      (field) => field.value.trim() === "",
    );
  }
  if ("links" in content) {
    return (content.links as { url: string }[]).every(
      (link) => link.url.trim() === "",
    );
  }
  return element.isEmpty;
}

/** The first field of an element, so arriving from search lands on something writable. */
export function firstCursor(element: AssetElement): string | null {
  if (!writesInPlace(element)) return null;
  const content = element.content as Record<string, unknown>;
  if ("text" in content) return `${element.id}:text`;
  if ("texts" in content) return `${element.id}:0:text`;
  if ("turns" in content) return `${element.id}:0:speaker`;
  if ("fields" in content) return `${element.id}:0:name`;
  if ("links" in content) return `${element.id}:0:label`;
  return null;
}
