import { expect, test } from "bun:test";
import { leavesIllarin } from "./post-link";

test("an address on another site leaves Illarin", () => {
  expect(leavesIllarin("https://example.com/notes")).toBe(true);
});

test("an address on Illarin's own hostname stays", () => {
  expect(leavesIllarin("https://localhost:8000/browse")).toBe(false);
});

test("an address on one of Illarin's other hostnames stays", () => {
  expect(leavesIllarin("https://blog.localhost/first-post")).toBe(false);
});

test("a mailto address is not opened away", () => {
  expect(leavesIllarin("mailto:someone@example.com")).toBe(false);
});

test("an address that cannot be read is treated as leaving", () => {
  expect(leavesIllarin("https://")).toBe(true);
});
