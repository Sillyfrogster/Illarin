import { expect, test } from "bun:test";
import type { IngestOperation } from "./api/query";
import { importStage } from "./import-stage";

const url = "/v1/ingests/00000000-0000-4000-8000-000000000001";
const operation = (
  status: IngestOperation["status"],
  rest: Partial<IngestOperation> = {},
): IngestOperation => ({
  id: "00000000-0000-4000-8000-000000000001",
  status,
  url,
  ...rest,
});

test("waits for a file before anything has been handed over", () => {
  expect(importStage(null, "")).toEqual({ at: "choosing" });
});

test("says the file is in hand before Illarin has opened it", () => {
  expect(importStage(operation("pending"), "")).toEqual({
    at: "reading",
    heading: "Your file is in hand",
  });
});

test("says Illarin is reading once it has started", () => {
  expect(importStage(operation("processing"), "")).toEqual({
    at: "reading",
    heading: "Reading your file",
  });
});

test("reports a lost connection over the reading it interrupted", () => {
  expect(
    importStage(operation("processing"), "The connection was interrupted."),
  ).toEqual({ at: "lost", message: "The connection was interrupted." });
});

test("carries the refusal Illarin gave for a file it would not take", () => {
  expect(
    importStage(
      operation("failed", {
        failure: {
          reason: "unsupported_format",
          message: "This is not a card.",
        },
      }),
      "",
    ),
  ).toEqual({ at: "refused", message: "This is not a card." });
});

test("names a refusal Illarin sent without one, so the page is never blank", () => {
  expect(importStage(operation("failed"), "")).toEqual({
    at: "refused",
    message: "Illarin could not read this file.",
  });
});

test("hands over the asset a finished import made", () => {
  const asset = {
    id: "00000000-0000-4000-8000-000000000002",
    name: "A quiet cartographer",
    kind: "character",
  } as NonNullable<IngestOperation["asset"]>;

  expect(importStage(operation("success", { asset }), "")).toEqual({
    at: "arrived",
    asset,
  });
});

test("keeps waiting where a status arrives without the asset it promised", () => {
  expect(importStage(operation("success"), "")).toEqual({
    at: "reading",
    heading: "Reading your file",
  });
});

test("treats a cancelled or previewed operation as one still being read", () => {
  expect(importStage(operation("cancelled"), "")).toEqual({
    at: "reading",
    heading: "Reading your file",
  });
  expect(importStage(operation("preview"), "")).toEqual({
    at: "reading",
    heading: "Reading your file",
  });
});

test("a refusal outranks a lost connection, because the refusal is the outcome", () => {
  expect(
    importStage(operation("failed"), "We lost the latest update."),
  ).toEqual({ at: "refused", message: "Illarin could not read this file." });
});
