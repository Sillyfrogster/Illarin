import type { IngestOperation, ReplacementDecision } from "@/lib/api/query";

const UNSETTLED = ["pending", "processing", "preview"];

export function updateStanding(
  waiting: IngestOperation | null,
  changed: boolean,
): string {
  if (waiting?.status === "preview") {
    return "An uploaded file is waiting for your review. Accept or discard it before you publish.";
  }
  if (waiting) {
    return "Illarin is reading the file you uploaded. Readers keep the published version while it works.";
  }
  if (changed) {
    return "You have changes readers do not have. Review them into an update when you are ready.";
  }
  return "Readers have everything on this page.";
}

export function reviewBlockedReason(
  waiting: IngestOperation | null,
  changed: boolean,
): string {
  if (waiting) {
    return "An uploaded file is waiting for your review. Accept or discard it before publishing.";
  }
  if (!changed) {
    return "Nothing has changed since the last update. Edit the page or replace the file first.";
  }
  return "";
}

export function replacementAction(
  operation: IngestOperation | null,
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
  operation: IngestOperation | null,
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

export function unsettledReplacement(
  operation: IngestOperation,
): IngestOperation | null {
  return UNSETTLED.includes(operation.status) ? operation : null;
}
