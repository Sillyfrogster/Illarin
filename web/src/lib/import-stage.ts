import type { IngestOperation } from "./api/query";

export type ImportedAsset = NonNullable<IngestOperation["asset"]>;

/** Where an import has got to, and the one thing the page says about it there. */
export type ImportStage =
  | { at: "choosing" }
  | { at: "reading"; heading: string }
  | { at: "lost"; message: string }
  | { at: "refused"; message: string }
  | { at: "arrived"; asset: ImportedAsset };

const STILL_READING = "Reading your file";

/** Reads the operation and the last connection trouble into the one state the page shows. */
export function importStage(
  operation: IngestOperation | null,
  message: string,
): ImportStage {
  if (!operation) return { at: "choosing" };

  if (operation.status === "failed") {
    return {
      at: "refused",
      message:
        operation.failure?.message ?? "Illarin could not read this file.",
    };
  }

  if (operation.status === "success" && operation.asset) {
    return { asset: operation.asset, at: "arrived" };
  }

  if (message) return { at: "lost", message };

  return {
    at: "reading",
    heading:
      operation.status === "pending" ? "Your file is in hand" : STILL_READING,
  };
}
