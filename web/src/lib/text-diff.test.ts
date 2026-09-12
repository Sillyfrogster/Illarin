import { expect, test } from "bun:test";
import { type DiffPiece, wordDiff } from "./text-diff";

function rejoin(pieces: DiffPiece[], kind: DiffPiece["kind"]): string {
  return pieces
    .filter((piece) => piece.kind !== kind)
    .map((piece) => piece.text)
    .join("");
}

test("keeps the words both sides share and marks the one that changed", () => {
  expect(wordDiff("a quiet archivist", "a loud archivist")).toEqual([
    { kind: "same", text: "a " },
    { kind: "removed", text: "quiet" },
    { kind: "added", text: "loud" },
    { kind: "same", text: " archivist" },
  ]);
});

test("reports an unchanged text as one unchanged piece", () => {
  expect(wordDiff("no change here", "no change here")).toEqual([
    { kind: "same", text: "no change here" },
  ]);
});

test("reads an empty side as the whole text arriving or leaving", () => {
  expect(wordDiff("", "brand new")).toEqual([
    { kind: "added", text: "brand new" },
  ]);
  expect(wordDiff("gone now", "")).toEqual([
    { kind: "removed", text: "gone now" },
  ]);
});

test("rebuilds each side from the pieces that belong to it", () => {
  const before = "She keeps the ledger in the back room, alone.";
  const after = "She keeps the ledger in the front room, mostly alone.";
  const pieces = wordDiff(before, after);

  expect(rejoin(pieces, "added")).toBe(before);
  expect(rejoin(pieces, "removed")).toBe(after);
});

test("reports wholly rewritten text as one removal and one addition", () => {
  const before = Array.from({ length: 700 }, (_, at) => `old${at}`).join(" ");
  const after = Array.from({ length: 700 }, (_, at) => `new${at}`).join(" ");

  expect(wordDiff(before, after)).toEqual([
    { kind: "removed", text: before },
    { kind: "added", text: after },
  ]);
});
