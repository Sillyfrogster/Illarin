import type { IngestOperation, ReplacementDecision } from "@/lib/api/query";

/** The statuses a replacement passes through before its creator has answered it. */
const UNSETTLED = ["pending", "processing", "preview"];

/** What the publication rail says about the version readers have. */
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

/** Why an update cannot be reviewed yet, and nothing where it can. */
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

/** What the replacement step's one action does in the state it is in. */
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

/** Whether that action has everything it needs. */
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

/** The operation the page still has to account for, and nothing once it is settled. */
export function unsettledReplacement(
  operation: IngestOperation,
): IngestOperation | null {
  return UNSETTLED.includes(operation.status) ? operation : null;
}
