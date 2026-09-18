import type { IngestOperation } from "./api/query";

export type ImportedWork = NonNullable<IngestOperation["work"]>;

export type ImportStage =
  | { at: "choosing" }
  | { at: "reading"; heading: string }
  | { at: "lost"; message: string }
  | { at: "refused"; message: string }
  | { at: "arrived"; work: ImportedWork };

const STILL_READING = "Reading your file";

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

  if (operation.status === "success" && operation.work) {
    return { work: operation.work, at: "arrived" };
  }

  if (message) return { at: "lost", message };

  return {
    at: "reading",
    heading: operation.status === "pending" ? "Upload received" : STILL_READING,
  };
}
