import { expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import {
  fromEditor,
  toEditor,
} from "@/components/publication/editor/tiptap-document";
import {
  isWritten,
  POST_DOCUMENT_VERSION,
  type PostDocument,
} from "@/lib/post-document";

/** The one corpus Go validation and the site both read. Go owns the document. */
const CORPUS = join(
  import.meta.dir,
  "../../../api/internal/postdoc/testdata/corpus",
);

type CorpusCase = {
  note: string;
  document: PostDocument;
  canonical?: PostDocument;
};

function corpus(group: string): [string, CorpusCase][] {
  const root = join(CORPUS, group);
  return readdirSync(root).map((name) => [
    name,
    JSON.parse(readFileSync(join(root, name), "utf8")) as CorpusCase,
  ]);
}

test("the corpus is there to read", () => {
  expect(corpus("valid").length).toBeGreaterThan(0);
});

for (const [name, one] of corpus("valid")) {
  test(`${name} survives a round trip through the editor`, () => {
    const canonical = one.canonical ?? one.document;
    expect(fromEditor(toEditor(canonical))).toEqual(canonical);
  });

  test(`${name} is written at the version this build emits`, () => {
    expect(fromEditor(toEditor(one.document)).version).toBe(
      POST_DOCUMENT_VERSION,
    );
  });
}

test("the editor drops a node outside the vocabulary rather than storing it", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      { type: "codeBlock", content: [{ type: "text", text: "rm -rf /" }] },
      { type: "paragraph", content: [{ type: "text", text: "Kept" }] },
    ],
  });
  expect(carried.content).toEqual([
    { type: "paragraph", content: [{ type: "text", text: "Kept" }] },
  ]);
});

test("the editor drops a mark outside the vocabulary", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "paragraph",
        content: [
          {
            type: "text",
            text: "Loud",
            marks: [{ type: "highlight" }, { type: "bold" }],
          },
        ],
      },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "paragraph",
      content: [{ type: "text", text: "Loud", marks: [{ type: "bold" }] }],
    },
  ]);
});

test("marks come back in the order the canonical document uses", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "paragraph",
        content: [
          {
            type: "text",
            text: "Both",
            marks: [
              { type: "link", attrs: { href: "https://example.com" } },
              { type: "italic" },
              { type: "bold" },
            ],
          },
        ],
      },
    ],
  });
  expect(carried.content[0]).toEqual({
    type: "paragraph",
    content: [
      {
        type: "text",
        text: "Both",
        marks: [
          { type: "bold" },
          { type: "italic" },
          { type: "link", href: "https://example.com" },
        ],
      },
    ],
  });
});

test("a body of whitespace does not count as writing", () => {
  expect(
    isWritten({
      version: POST_DOCUMENT_VERSION,
      content: [
        { type: "paragraph", content: [{ type: "text", text: "   " }] },
      ],
    }),
  ).toBe(false);
});
