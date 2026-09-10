import { expect, test } from "bun:test";
import type { ReadinessItem } from "@/lib/api/query";
import { readinessTarget } from "./readiness";

function item(over: Partial<ReadinessItem>): ReadinessItem {
  return { detail: "", id: "name", label: "Name", met: false, ...over };
}

test("sends a block requirement to the block that holds it", () => {
  const blockId = "1c9d4e77-2a53-4b8f-9e01-5d6a7b8c9012";

  expect(readinessTarget(item({ blockId, id: "greetings" }))).toEqual({
    blockId,
    where: "block",
  });
});

test("sends the two header requirements to where the page writes them", () => {
  expect(readinessTarget(item({ id: "name" }))).toEqual({ where: "name" });
  expect(readinessTarget(item({ id: "adult_content" }))).toEqual({
    where: "rating",
  });
});

test("sends an unanswered upload to the replacement it is waiting on", () => {
  expect(readinessTarget(item({ id: "upload" }))).toEqual({
    where: "replacement",
  });
});

test("offers nowhere to go for a requirement no control answers", () => {
  expect(readinessTarget(item({ id: "export" }))).toBeNull();
  expect(readinessTarget(item({ id: "media" }))).toBeNull();
});
