import { describe, expect, test } from "bun:test";
import { clauseAnchor, LEGAL_DOCUMENTS, nextDocument } from "./legal-documents";

describe("a clause anchor", () => {
  test("drops the number a reader sees and keeps the words", () => {
    expect(clauseAnchor("1. Who we are")).toBe("who-we-are");
    expect(clauseAnchor("15. Limit of liability")).toBe("limit-of-liability");
  });

  test("survives punctuation, case and repeated spaces", () => {
    expect(clauseAnchor("4.2 What you let us do with it")).toBe(
      "what-you-let-us-do-with-it",
    );
    expect(clauseAnchor("DMCA  &  copyright")).toBe("dmca-copyright");
  });

  test("gives every clause of one document its own anchor", () => {
    const headings = ["1. Accounts", "2. Accounts and you", "3. Contact"];
    expect(new Set(headings.map(clauseAnchor)).size).toBe(3);
  });
});

describe("reading on", () => {
  test("offers the next document in the order they are read", () => {
    expect(nextDocument("/legal/terms")).toEqual(LEGAL_DOCUMENTS[1]);
    expect(nextDocument("/legal/acceptable-use")).toEqual(LEGAL_DOCUMENTS[3]);
  });

  test("offers nothing after the last one", () => {
    expect(nextDocument("/legal/dmca")).toBeNull();
  });
});
