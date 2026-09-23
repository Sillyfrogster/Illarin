import { expect, test } from "bun:test";
import { addressOn } from "./address";

const ORIGIN = "https://example.test";

test("a path and query land on the origin", () => {
  expect(addressOn(ORIGIN, "/")).toBe("https://example.test/");
  expect(addressOn(ORIGIN, "/first-post", "?ref=x")).toBe(
    "https://example.test/first-post?ref=x",
  );
  expect(addressOn(ORIGIN, "/caf%C3%A9/card.png")).toBe(
    "https://example.test/caf%C3%A9/card.png",
  );
});

test("a path that reads like another host stays a path on this origin", () => {
  for (const path of ["//evil.com", "/\\evil.com", "//"]) {
    expect(new URL(addressOn(ORIGIN, path, "?x=1")).host).toBe("example.test");
  }
});
