import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { PostContentsEntry } from "@/lib/post-contents";
import { ArticleContents } from "./ArticleContents";
import { ShareArticle } from "./ShareArticle";

const ENTRIES: PostContentsEntry[] = [
  { anchor: "what-is-stored", label: "What is stored", level: 2 },
  { anchor: "what-is-refused", label: "What is refused", level: 2 },
  { anchor: "unknown-nodes", label: "Unknown nodes", level: 3 },
];

const PERMALINK = "http://localhost:8000/blog/first-post";

test("the contents reach every heading by its own address", () => {
  const html = renderToStaticMarkup(<ArticleContents entries={ENTRIES} />);
  for (const entry of ENTRIES) {
    expect(html).toContain(`href="#${entry.anchor}"`);
    expect(html).toContain(entry.label);
  }
});

test("the contents are a named navigation with a disclosure a narrow screen can use", () => {
  const html = renderToStaticMarkup(<ArticleContents entries={ENTRIES} />);
  expect(html).toContain('aria-label="Contents"');
  expect(html).toContain('aria-controls="article-contents"');
  expect(html).toContain('aria-expanded="false"');
  expect(html).toContain('id="article-contents"');
});

test("a deeper heading is marked as one rather than flattened", () => {
  const html = renderToStaticMarkup(<ArticleContents entries={ENTRIES} />);
  expect(html).toContain('data-level="3"');
});

test("sharing offers copy-link before the browser has said anything", () => {
  const html = renderToStaticMarkup(
    <ShareArticle permalink={PERMALINK} title="First post" />,
  );
  expect(html).toContain("Copy link");
  expect(html).toContain("<output");
});

test("no share sheet is offered until the device is known to carry one", () => {
  const html = renderToStaticMarkup(
    <ShareArticle permalink={PERMALINK} title="First post" />,
  );
  expect(html.match(/<button/g)).toHaveLength(1);
});

test("nothing about sharing reaches a network", () => {
  const html = renderToStaticMarkup(
    <ShareArticle permalink={PERMALINK} title="First post" />,
  );
  expect(html).not.toContain("iframe");
  expect(html).not.toContain("http://localhost:8000/blog/first-post");
});
