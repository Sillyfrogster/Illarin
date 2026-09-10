import { expect, test } from "bun:test";
import type { IngestOperation } from "@/lib/api/query";
import {
  replacementAction,
  replacementReady,
  reviewBlockedReason,
  unsettledReplacement,
  updateStanding,
} from "./asset-publication";

function operation(over: Partial<IngestOperation>): IngestOperation {
  return {
    id: "3b8f1e2a-77c4-4d19-8a52-6c0e9f4b1d33",
    status: "pending",
    url: "/v1/ingest/3b8f1e2a-77c4-4d19-8a52-6c0e9f4b1d33",
    ...over,
  };
}

const previewing = operation({
  preview: { changes: [], format: "character card v2", unrepresentable: [] },
  status: "preview",
});

test("says a reviewable upload is waiting before anything else", () => {
  expect(updateStanding(previewing, true)).toBe(
    "An uploaded file is waiting for your review. Accept or discard it before you publish.",
  );
});

test("says a file is still being read while it is processing", () => {
  expect(updateStanding(operation({ status: "processing" }), false)).toBe(
    "Illarin is reading the file you uploaded. Readers keep the published version while it works.",
  );
});

test("separates changes readers do not have from a page they already have", () => {
  expect(updateStanding(null, true)).toBe(
    "You have changes readers do not have. Review them into an update when you are ready.",
  );
  expect(updateStanding(null, false)).toBe(
    "Readers have everything on this page.",
  );
});

test("refuses a review while an upload is unanswered or nothing has changed", () => {
  expect(reviewBlockedReason(previewing, true)).toBe(
    "An uploaded file is waiting for your review. Accept or discard it before publishing.",
  );
  expect(reviewBlockedReason(null, false)).toBe(
    "Nothing has changed since the last update. Edit the page or replace the file first.",
  );
  expect(reviewBlockedReason(null, true)).toBe("");
});

test("names what the replacement action does in the state it is in", () => {
  expect(replacementAction(null, false)).toBe("Upload this file");
  expect(replacementAction(null, true)).toBe("Uploading…");
  expect(replacementAction(operation({ status: "processing" }), false)).toBe(
    "Reading…",
  );
  expect(replacementAction(previewing, false)).toBe("Apply this file");
  expect(replacementAction(previewing, true)).toBe("Applying…");
  expect(replacementAction(operation({ status: "failed" }), false)).toBe(
    "Choose another file",
  );
});

test("waits for every keep or remove decision before a file can be applied", () => {
  const deciding = operation({
    preview: {
      changes: [],
      format: "character card v1",
      unrepresentable: ["group_only_greetings", "images"],
    },
    status: "preview",
  });

  expect(replacementReady(deciding, null, {})).toBe(false);
  expect(
    replacementReady(deciding, null, { group_only_greetings: "keep" }),
  ).toBe(false);
  expect(
    replacementReady(deciding, null, {
      group_only_greetings: "keep",
      images: "remove",
    }),
  ).toBe(true);
});

test("needs a chosen file before an upload and nothing after a failure", () => {
  const file = new File(["card"], "card.png");

  expect(replacementReady(null, null, {})).toBe(false);
  expect(replacementReady(null, file, {})).toBe(true);
  expect(replacementReady(operation({ status: "failed" }), null, {})).toBe(
    true,
  );
});

test("keeps an unsettled operation and drops one the creator has finished with", () => {
  expect(unsettledReplacement(previewing)).toBe(previewing);
  expect(
    unsettledReplacement(operation({ status: "processing" }))?.status,
  ).toBe("processing");
  expect(unsettledReplacement(operation({ status: "cancelled" }))).toBeNull();
  expect(unsettledReplacement(operation({ status: "success" }))).toBeNull();
  expect(unsettledReplacement(operation({ status: "failed" }))).toBeNull();
});
