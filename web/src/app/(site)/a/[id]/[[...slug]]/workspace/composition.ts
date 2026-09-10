import type {
  AddableBlock,
  ArrangeAssetBlocksRequest,
  AssetBlock,
  AssetElement,
} from "@/lib/api/query";
import { type BlockLayout, LAYOUTS } from "@/lib/page-arrangement";

export type BlockDestination = { position: number; label: string };

export type BlockOffer = { addable: AddableBlock; alreadyOn: boolean };

export type OfferGroup = { key: string; title: string; offers: BlockOffer[] };

export function moveBlock(
  blocks: AssetBlock[],
  blockId: string,
  to: number,
): AssetBlock[] {
  const from = blocks.findIndex((block) => block.id === blockId);
  if (from < 0 || to === from || to < 0 || to >= blocks.length) return blocks;
  const next = [...blocks];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next.map((block, position) => ({ ...block, position }));
}

export function blockDestinations(
  blocks: AssetBlock[],
  blockId: string,
): BlockDestination[] {
  const from = blocks.findIndex((block) => block.id === blockId);
  if (from < 0 || blocks.length < 2) return [];
  const rest = blocks.filter((_, index) => index !== from);
  return blocks
    .map((_, position) => position)
    .filter((position) => position !== from)
    .map((position) => ({
      label:
        position < rest.length
          ? `Before “${rest[position].title}”`
          : `After “${rest[rest.length - 1].title}”`,
      position,
    }));
}

export function seatElements(
  layout: BlockLayout,
  elements: AssetElement[],
): AssetElement[] {
  const slots = LAYOUTS[layout].slots;
  return elements.map((element, index) => {
    const slot = slots[Math.min(index, slots.length - 1)];
    return element.slot === slot ? element : { ...element, slot };
  });
}

export function relaidBlock(
  block: AssetBlock,
  layout: BlockLayout,
): AssetBlock {
  return { ...block, elements: seatElements(layout, block.elements), layout };
}

export function moveElement(
  block: AssetBlock,
  elementId: string,
  to: number,
): AssetBlock {
  const elements = block.elements;
  const from = elements.findIndex((element) => element.id === elementId);
  if (from < 0 || to === from || to < 0 || to >= elements.length) return block;
  const next = [...elements];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return { ...block, elements: seatElements(block.layout, next) };
}

export function removeElement(
  block: AssetBlock,
  elementId: string,
): AssetBlock {
  const dropped = block.elements.find((element) => element.id === elementId);
  if (!dropped || dropped.pinned) return block;
  return {
    ...block,
    elements: seatElements(
      block.layout,
      block.elements.filter((element) => element.id !== elementId),
    ),
  };
}

export function offerGroups(
  addable: AddableBlock[],
  blocks: AssetBlock[],
  search: string,
): OfferGroup[] {
  const onThePage = new Set(blocks.map((block) => block.definition));
  const wanted = search.trim().toLowerCase();
  const groups = new Map<string, OfferGroup>();
  for (const candidate of addable) {
    if (
      wanted &&
      !candidate.title.toLowerCase().includes(wanted) &&
      !candidate.summary.toLowerCase().includes(wanted)
    ) {
      continue;
    }
    const offer: BlockOffer = {
      addable: candidate,
      alreadyOn: !candidate.repeatable && onThePage.has(candidate.definition),
    };
    const group = groups.get(candidate.group);
    if (group) group.offers.push(offer);
    else {
      groups.set(candidate.group, {
        key: candidate.group,
        offers: [offer],
        title: candidate.groupTitle,
      });
    }
  }
  return [...groups.values()];
}

export function arrangementRequest(
  order: AssetBlock[],
  saved: AssetBlock[],
): ArrangeAssetBlocksRequest {
  const widths = new Map(saved.map((block) => [block.id, block.width]));
  return {
    blocks: order.map((block) => ({
      hidden: block.hidden,
      id: block.id,
      width: widths.get(block.id) ?? block.width,
    })),
  };
}
