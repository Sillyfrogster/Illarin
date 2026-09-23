import { expect, test } from "bun:test";
import { profilePath, readProfileAddress } from "./profile-address";

test("reads the handle from the address", () => {
  expect(readProfileAddress("@garden.keeper")).toBe("garden.keeper");
});

test("is no address without the at sign", () => {
  expect(readProfileAddress("garden.keeper")).toBe(null);
});

test("lowercases the handle", () => {
  expect(readProfileAddress("@Garden.Keeper")).toBe("garden.keeper");
});

test("reads an all-digit handle", () => {
  expect(readProfileAddress("@314159")).toBe("314159");
});

test("is no address at all when the handle could not exist", () => {
  expect(readProfileAddress("@")).toBe(null);
  expect(readProfileAddress("")).toBe(null);
  expect(readProfileAddress("@no")).toBe(null);
  expect(readProfileAddress(`@${"a".repeat(33)}`)).toBe(null);
  expect(readProfileAddress("@garden keeper")).toBe(null);
  expect(readProfileAddress("@...")).toBe(null);
});

test("a profile lives at its handle", () => {
  expect(profilePath("garden.keeper")).toBe("/@garden.keeper");
});
