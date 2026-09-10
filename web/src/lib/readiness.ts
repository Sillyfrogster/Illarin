import type { ReadinessItem } from "@/lib/api/query";

/** Where a creator goes to meet one requirement, and nothing where no control answers it. */
export type ReadinessTarget =
  | { where: "block"; blockId: string }
  | { where: "name" }
  | { where: "rating" }
  | { where: "replacement" };

/** The targets that are somewhere on the page, rather than a step of the rail. */
export type PageTarget = Exclude<ReadinessTarget, { where: "replacement" }>;

export function readinessTarget(item: ReadinessItem): ReadinessTarget | null {
  if (item.blockId) return { blockId: item.blockId, where: "block" };
  if (item.id === "name") return { where: "name" };
  if (item.id === "adult_content") return { where: "rating" };
  if (item.id === "upload") return { where: "replacement" };
  return null;
}
