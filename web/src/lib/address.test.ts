import { expect, test } from "bun:test";
import { addressOn } from "./address";

const ORIGIN = "http://blog.localhost:8000";

test("a path and query land on the origin", () => {
  expect(addressOn(ORIGIN, "/")).toBe("http://blog.localhost:8000/");
  expect(addressOn(ORIGIN, "/first-post", "?ref=x")).toBe(
    "http://blog.localhost:8000/first-post?ref=x",
  );
  expect(addressOn(ORIGIN, "/caf%C3%A9/card.png")).toBe(
    "http://blog.localhost:8000/caf%C3%A9/card.png",
  );
});

test("a path that reads like another host stays a path on this origin", () => {
  for (const path of ["//evil.com", "/\\evil.com", "//"]) {
    expect(new URL(addressOn(ORIGIN, path, "?x=1")).host).toBe(
      "blog.localhost:8000",
    );
  }
});
