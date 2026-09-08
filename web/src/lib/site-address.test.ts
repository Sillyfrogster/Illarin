import { expect, test } from "bun:test";
import { siteAddress } from "./site-address";

test("a page on Illarin is named by its whole address, because the blog answers elsewhere", () => {
  expect(siteAddress("/")).toBe("http://localhost:8000/");
  expect(siteAddress("/browse")).toBe("http://localhost:8000/browse");
});

test("a legal document keeps the path it was given", () => {
  expect(siteAddress("/terms")).toBe("http://localhost:8000/terms");
});
