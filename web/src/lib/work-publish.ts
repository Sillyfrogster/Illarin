import type { ReplacementDecision, UploadOperation } from "@/lib/api/query";

const UNSETTLED = ["pending", "processing", "preview"];

export function updateStanding(
  waiting: UploadOperation | null,
  changed: boolean,
): string {
  if (waiting?.status === "preview") {
    return "An uploaded file is waiting for your review. Accept or discard it before you publish.";
  }
  if (waiting) {
    return "Illarin is reading the file you uploaded. Readers keep the published version while it works.";
  }
  if (changed) {
    return "You have drafted changes. Review them before publishing a version.";
  }
  return "All changes are published.";
}

export function reviewBlockedReason(
  waiting: UploadOperation | null,
  changed: boolean,
): string {
  if (waiting) {
    return "An uploaded file is waiting for your review. Accept or discard it before publishing.";
  }
  if (!changed) {
    return "Nothing has changed since the last version. Edit the page or replace the file first.";
  }
  return "";
}

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
