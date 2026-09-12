import type { ReadinessItem } from "@/lib/api/query";

export type ReadinessTarget =
  | { where: "block"; blockId: string }
  | { where: "name" }
  | { where: "rating" }
  | { where: "replacement" };

export type PageTarget = Exclude<ReadinessTarget, { where: "replacement" }>;

export function readinessTarget(item: ReadinessItem): ReadinessTarget | null {
  if (item.blockId) return { blockId: item.blockId, where: "block" };
  if (item.id === "name") return { where: "name" };
  if (item.id === "adult_content") return { where: "rating" };
  if (item.id === "upload") return { where: "replacement" };
  return null;
}
