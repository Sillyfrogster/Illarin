import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { PostByline } from "@/lib/api/query";
import { Byline, BylineText } from "./Byline";

const WREN: PostByline = {
  handle: "wren",
  displayName: "Wren Ashdown",
  contactEmail: "",
  historical: false,
  positions: ["Founder"],
  distinctions: [],
};

const KEPT: PostByline = { ...WREN, historical: true };

test("a name with an account behind it reaches that profile on the main origin", () => {
  const html = renderToStaticMarkup(<Byline byline={WREN} />);
  expect(html).toContain('href="http://localhost:8000/@wren"');
  expect(html).toContain("Wren Ashdown");
});

test("a name kept from before, with no account, is not a link at all", () => {
  const html = renderToStaticMarkup(<Byline byline={KEPT} />);
  expect(html).not.toContain("href=");
  expect(html).toContain("Wren Ashdown");
});

test("a listed name is words rather than a second action inside its row", () => {
  const html = renderToStaticMarkup(<BylineText affiliation byline={WREN} />);
  expect(html).toContain("Wren Ashdown");
  expect(html).toContain("Illarin Team");
  expect(html).not.toContain("href=");
});

test("a listed name drops the affiliation its archive already states", () => {
  const html = renderToStaticMarkup(
    <BylineText affiliation={false} byline={KEPT} />,
  );
  expect(html).toContain("Wren Ashdown");
  expect(html).not.toContain("Illarin Team");
});

test("a byline with no display name is attributed to its handle", () => {
  const html = renderToStaticMarkup(
    <Byline byline={{ ...WREN, displayName: "" }} />,
  );
  expect(html).toContain("@wren");
});
