import type { ReplacementDecision, UploadOperation } from "@/lib/api/query";

const UNSETTLED = ["pending", "processing", "preview"];

export function replacementAction(
  operation: UploadOperation | null,
  busy: boolean,
): string {
  if (operation?.status === "preview") {
    return busy ? "Applying…" : "Apply this file";
  }
  if (operation?.status === "failed") return "Choose another file";
  if (operation) return "Reading…";
  return busy ? "Uploading…" : "Upload this file";
}

export function replacementReady(
  operation: UploadOperation | null,
  file: File | null,
  decisions: ReplacementDecision,
): boolean {
  if (operation?.status === "preview") {
    return (operation.preview?.unrepresentable ?? []).every(
      (role) => decisions[role] !== undefined,
    );
  }
  if (operation?.status === "failed") return true;
  return !operation && file !== null;
}

export function previewConflicts(preview: {
  conflicts?: string[] | null;
}): string[] {
  return preview.conflicts ?? [];
}

export function unsettledReplacement(
  operation: UploadOperation,
): UploadOperation | null {
  return UNSETTLED.includes(operation.status) ? operation : null;
}
