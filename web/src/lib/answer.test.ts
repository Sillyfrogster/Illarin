import { expect, test } from "bun:test";
import { isStringArray, refusalMessage } from "./answer";

test("repeats the reason Illarin gave for a refusal", () => {
  expect(refusalMessage({ error: "That code has expired." }, "fallback")).toBe(
    "That code has expired.",
  );
});

test("falls back when Illarin gave no reason", () => {
  expect(refusalMessage({ error: "  " }, "fallback")).toBe("fallback");
  expect(refusalMessage({}, "fallback")).toBe("fallback");
  expect(refusalMessage(null, "fallback")).toBe("fallback");
  expect(refusalMessage("not an object", "fallback")).toBe("fallback");
});

test("reads a list of names, and refuses anything else", () => {
  expect(isStringArray(["one", "two"])).toBe(true);
  expect(isStringArray([])).toBe(true);
  expect(isStringArray(["one", 2])).toBe(false);
  expect(isStringArray("one")).toBe(false);
  expect(isStringArray(null)).toBe(false);
});
