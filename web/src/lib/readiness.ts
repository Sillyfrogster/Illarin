import type { ReadinessItem } from "@/lib/api/query";

/** Where a creator goes to meet one requirement. */
export function readinessHref(item: ReadinessItem): string {
  if (item.blockId) return `#block-${item.blockId}`;
  return item.id === "adult_content" ? "#adult-content-answer" : "#asset-name";
}
