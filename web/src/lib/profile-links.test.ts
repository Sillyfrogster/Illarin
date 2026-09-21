import { expect, test } from "bun:test";
import { linkBrand, linkText } from "./profile-links";

test("recognises the sites creators link to, with or without www and subdomains", () => {
  expect(linkBrand("https://discord.gg/abc")).toBe("discord");
  expect(linkBrand("https://www.ko-fi.com/wren")).toBe("ko-fi");
  expect(linkBrand("https://twitter.com/wren")).toBe("x");
  expect(linkBrand("https://wren.itch.io")).toBe("itch");
  expect(linkBrand("https://example.com/notes")).toBeNull();
  expect(linkBrand("not a link")).toBeNull();
});

test("shows a link as its host and path, shortened when long", () => {
  expect(linkText("https://www.miravale.example/")).toBe("miravale.example");
  expect(linkText("https://ko-fi.com/wren")).toBe("ko-fi.com/wren");
  expect(
    linkText("https://example.com/a/very/long/path/that/keeps/going/on/and/on"),
  ).toBe("example.com/a/very/long/path/that/keeps…");
});
