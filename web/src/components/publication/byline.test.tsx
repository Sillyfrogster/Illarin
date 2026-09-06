import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { PostByline } from "@/lib/api/query";
import { Byline, BylineLine } from "./Byline";

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

test("a listed name follows the same rule as a full byline", () => {
  expect(renderToStaticMarkup(<BylineLine byline={WREN} />)).toContain(
    'href="http://localhost:8000/@wren"',
  );
  expect(
    renderToStaticMarkup(<BylineLine byline={KEPT} quiet />),
  ).not.toContain("href=");
});

test("a byline with no display name is attributed to its handle", () => {
  const html = renderToStaticMarkup(
    <Byline byline={{ ...WREN, displayName: "" }} />,
  );
  expect(html).toContain("@wren");
});
