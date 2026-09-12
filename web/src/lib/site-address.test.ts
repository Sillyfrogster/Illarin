import { expect, test } from "bun:test";
import type { PostByline } from "@/lib/api/query";
import { bylineProfile, profileAddress, siteAddress } from "./site-address";

const BYLINE: PostByline = {
  handle: "wren",
  displayName: "Wren Ashdown",
  contactEmail: "",
  historical: false,
};

test("a site address is whole, so it holds wherever it is linked from", () => {
  expect(siteAddress("/browse")).toBe("http://localhost:8000/browse");
});

test("a profile address is on the main origin", () => {
  expect(profileAddress("garden.keeper")).toBe(
    "http://localhost:8000/@garden.keeper",
  );
});

test("a byline links its profile unless no account stands behind it", () => {
  expect(bylineProfile(BYLINE)).toBe("http://localhost:8000/@wren");
  expect(bylineProfile({ ...BYLINE, historical: true })).toBeNull();
});
