import { expect, test } from "bun:test";
import { isSafeAddress, leavesIllarin } from "./post-link";

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

test("only an https or mailto address is safe", () => {
  expect(isSafeAddress("https://example.com/notes")).toBe(true);
  expect(isSafeAddress("mailto:someone@example.com")).toBe(true);
  expect(isSafeAddress("javascript:alert(1)")).toBe(false);
  expect(isSafeAddress("data:text/html,<b>x</b>")).toBe(false);
  expect(isSafeAddress("http://example.com")).toBe(false);
  expect(isSafeAddress("https://")).toBe(false);
  expect(isSafeAddress("https://exa mple.com")).toBe(false);
});
