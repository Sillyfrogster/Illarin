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
      { type: "iframe", attrs: { src: "https://example.com" } },
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

test("a line the author has not typed into stays out of the working copy", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      { type: "paragraph", content: [{ type: "text", text: "Written" }] },
      { type: "heading", attrs: { level: 2 }, content: [] },
      { type: "paragraph" },
    ],
  });
  expect(carried.content).toEqual([
    { type: "paragraph", content: [{ type: "text", text: "Written" }] },
  ]);
});

test("an empty list item and its list stay out of the working copy", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "bulletList",
        content: [
          {
            type: "listItem",
            content: [
              { type: "paragraph", content: [{ type: "text", text: "One" }] },
            ],
          },
          { type: "listItem", content: [{ type: "paragraph" }] },
        ],
      },
      { type: "blockquote", content: [{ type: "paragraph" }] },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "bulletList",
      content: [
        {
          type: "listItem",
          content: [
            { type: "paragraph", content: [{ type: "text", text: "One" }] },
          ],
        },
      ],
    },
  ]);
});

test("a heading keeps the address it read and a new one waits for Go to write it", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "heading",
        attrs: { level: 2, anchor: "release-notes" },
        content: [{ type: "text", text: "Renamed since" }],
      },
      {
        type: "heading",
        attrs: { level: 3, anchor: null },
        content: [{ type: "text", text: "Brand new" }],
      },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "heading",
      level: 2,
      anchor: "release-notes",
      content: [{ type: "text", text: "Renamed since" }],
    },
    {
      type: "heading",
      level: 3,
      content: [{ type: "text", text: "Brand new" }],
    },
  ]);
});

test("a task keeps its state and an untyped task stays out", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "taskList",
        content: [
          {
            type: "taskItem",
            attrs: { checked: true },
            content: [
              { type: "paragraph", content: [{ type: "text", text: "Done" }] },
            ],
          },
          {
            type: "taskItem",
            attrs: { checked: false },
            content: [{ type: "paragraph" }],
          },
        ],
      },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "taskList",
      content: [
        {
          type: "taskItem",
          done: true,
          content: [
            { type: "paragraph", content: [{ type: "text", text: "Done" }] },
          ],
        },
      ],
    },
  ]);
});

test("a code block keeps its source and falls back to plain for an unknown label", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "codeBlock",
        attrs: { language: "brainfuck" },
        content: [{ type: "text", text: "one := 1\n" }],
      },
      { type: "codeBlock", attrs: { language: "go" }, content: [] },
    ],
  });
  expect(carried.content).toEqual([
    { type: "codeBlock", language: "plain", source: "one := 1\n" },
  ]);
});

test("a pasted table is squared off before it is stored", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "table",
        content: [
          {
            type: "tableRow",
            content: [
              {
                type: "tableHeader",
                attrs: { colspan: 2 },
                content: [
                  {
                    type: "paragraph",
                    content: [{ type: "text", text: "Both" }],
                  },
                ],
              },
            ],
          },
          {
            type: "tableRow",
            content: [
              {
                type: "tableCell",
                content: [
                  {
                    type: "paragraph",
                    content: [{ type: "text", text: "One" }],
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "table",
      content: [
        {
          type: "tableRow",
          content: [
            {
              type: "tableCell",
              heading: true,
              content: [
                {
                  type: "paragraph",
                  content: [{ type: "text", text: "Both" }],
                },
              ],
            },
            { type: "tableCell", heading: true, content: [] },
          ],
        },
        {
          type: "tableRow",
          content: [
            {
              type: "tableCell",
              content: [
                { type: "paragraph", content: [{ type: "text", text: "One" }] },
              ],
            },
            { type: "tableCell", content: [] },
          ],
        },
      ],
    },
  ]);
});

test("an inserted table and callout survive before anything is typed into them", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "table",
        content: [
          {
            type: "tableRow",
            content: [
              { type: "tableCell", content: [{ type: "paragraph" }] },
              { type: "tableCell", content: [{ type: "paragraph" }] },
            ],
          },
        ],
      },
      {
        type: "callout",
        attrs: { kind: "tip" },
        content: [{ type: "paragraph" }],
      },
    ],
  });
  expect(carried.content).toEqual([
    {
      type: "table",
      content: [
        {
          type: "tableRow",
          content: [
            { type: "tableCell", content: [] },
            { type: "tableCell", content: [] },
          ],
        },
      ],
    },
    {
      type: "callout",
      kind: "tip",
      content: [{ type: "paragraph", content: [] }],
    },
  ]);
});
