import type {
  SaveWorkBlockRequest,
  WorkBlock,
  WorkElement,
} from "@/lib/api/query";
import { writesInPlace as writtenInPlace } from "@/lib/page-arrangement";
import type { AllowedApp } from "../SealedPolicy";

export function writesInPlace(element: WorkElement): boolean {
  return writtenInPlace(element.type);
}

export function blockSaveRequest(
  block: WorkBlock,
  changes: {
    title?: string | null;
    layout?: WorkBlock["layout"];
    width?: WorkBlock["width"];
    elements?: WorkElement[];
    allowedApps?: AllowedApp[];
    exposeProtected?: boolean;
  } = {},
): SaveWorkBlockRequest {
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
  blocks: WorkBlock[],
  saved: WorkBlock,
): WorkBlock[] {
  return blocks.map((block) => (block.id === saved.id ? saved : block));
}

export function replaceElement(
  block: WorkBlock,
  element: WorkElement,
): WorkBlock {
  return {
    ...block,
    elements: block.elements.map((item) =>
      item.id === element.id ? element : item,
    ),
  };
}

export function changedBlockIds(
  draft: WorkBlock[],
  saved: WorkBlock[],
): string[] {
  const before = new Map(saved.map((block) => [block.id, block]));
  return draft
    .filter((block) => {
      const original = before.get(block.id);
      return original ? !sameContent(original, block) : true;
    })
    .map((block) => block.id);
}

function sameContent(left: WorkBlock, right: WorkBlock): boolean {
  return (
    left.title === right.title &&
    left.layout === right.layout &&
    left.width === right.width &&
    JSON.stringify(left.elements.map(contentOf)) ===
      JSON.stringify(right.elements.map(contentOf))
  );
}

function contentOf(element: WorkElement) {
  return {
    id: element.id,
    display: element.display,
    itemSize: element.itemSize,
    content: element.content,
  };
}

export function isEmptyContent(element: WorkElement): boolean {
  const content = element.content;
  if ("text" in content) return String(content.text ?? "").trim() === "";
  if ("texts" in content) {
    return content.texts.every((item) => item.text.trim() === "");
  }
  if ("turns" in content) {
    return content.turns.every((turn) => turn.text.trim() === "");
  }
  if ("fields" in content) {
    return content.fields.every((field) => field.value.trim() === "");
  }
  if ("links" in content) {
    return content.links.every((link) => link.url.trim() === "");
  }
  return element.isEmpty;
}

export function firstCursor(element: WorkElement): string | null {
  if (!writesInPlace(element)) return null;
  const content = element.content;
  if ("text" in content) return `${element.id}:text`;
  if ("texts" in content) return `${element.id}:0:text`;
  if ("turns" in content) return `${element.id}:0:speaker`;
  if ("fields" in content) return `${element.id}:0:name`;
  if ("links" in content) return `${element.id}:0:label`;
  return null;
}
