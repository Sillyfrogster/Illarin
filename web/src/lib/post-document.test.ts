import { expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import {
  fromEditor,
  toEditor,
} from "@/components/publication/writing/tiptap-document";
import {
  isPostAnchor,
  isWritten,
  POST_CALLOUT_KINDS,
  POST_DOCUMENT_VERSION,
  POST_LANGUAGES,
  type PostBlock,
  type PostDocument,
  type PostSpan,
} from "@/lib/post-document";
import { isSafeAddress } from "@/lib/post-link";

const CORPUS = join(
  import.meta.dir,
  "../../../api/internal/blog/body/testdata/corpus",
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
    expect(fromEditor(toEditor(canonical, []))).toEqual(canonical);
  });

  test(`${name} is written at the version this build emits`, () => {
    expect(fromEditor(toEditor(one.document, [])).version).toBe(
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

test("a line the author has not typed into stays out of the drafted changes", () => {
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

test("an empty list item and its list stay out of the drafted changes", () => {
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

for (const [name, one] of corpus("invalid")) {
  test(`${name} loses what Go refuses before it reaches the drafted changes`, () => {
    const carried = fromEditor({
      type: "doc",
      content: one.document.content,
    } as unknown as Parameters<typeof fromEditor>[0]);
    expect(carried.version).toBe(POST_DOCUMENT_VERSION);
    walk(carried.content);
  });
}

const BLOCKS = new Set([
  "paragraph",
  "heading",
  "bulletList",
  "orderedList",
  "taskList",
  "quote",
  "codeBlock",
  "table",
  "callout",
  "divider",
]);

const MARKS = new Set(["bold", "italic", "strike", "code", "link"]);

function walk(blocks: PostBlock[]) {
  for (const block of blocks) {
    expect(BLOCKS).toContain(block.type);
    switch (block.type) {
      case "paragraph":
        checkSpans(block.content);
        break;
      case "heading":
        expect(block.level).toBeGreaterThanOrEqual(2);
        expect(block.level).toBeLessThanOrEqual(4);
        if (block.anchor !== undefined) {
          expect(isPostAnchor(block.anchor)).toBe(true);
        }
        checkSpans(block.content);
        break;
      case "bulletList":
      case "orderedList":
      case "taskList":
        for (const entry of block.content) walk(entry.content);
        break;
      case "quote":
      case "callout":
        if (block.type === "callout") {
          expect(POST_CALLOUT_KINDS).toContain(block.kind);
        }
        walk(block.content);
        break;
      case "codeBlock":
        expect(POST_LANGUAGES).toContain(block.language);
        expect(block.source.trim().length).toBeGreaterThan(0);
        break;
      case "table": {
        const width = block.content[0]?.content.length ?? 0;
        for (const row of block.content) {
          expect(row.content.length).toBe(width);
          for (const cell of row.content) walk(cell.content);
        }
        break;
      }
    }
  }
}

function checkSpans(spans: PostSpan[]) {
  for (const span of spans) {
    for (const mark of span.marks ?? []) {
      expect(MARKS).toContain(mark.type);
      if (mark.type === "link") expect(isSafeAddress(mark.href)).toBe(true);
    }
  }
}

const PLACED = "6f2c1b40-9d38-4a7e-b512-0c8e37a41d59";

const HELD = [
  {
    id: PLACED,
    postId: "b8d0f1a2-3c45-4e67-89ab-cdef01234567",
    purpose: "document" as const,
    url: `/media/${PLACED}/detail/1`,
    thumbUrl: `/media/${PLACED}/grid/1`,
    width: 1200,
    height: 600,
  },
];

test("the editor draws a picture from the address Go answered with", () => {
  const document: PostDocument = {
    version: POST_DOCUMENT_VERSION,
    content: [
      {
        type: "image",
        mediaId: PLACED,
        alt: "The workspace",
        caption: "One draft.",
      },
    ],
  };
  const editing = toEditor(document, HELD);
  expect(editing.content?.[0]?.attrs).toMatchObject({
    mediaId: PLACED,
    src: `/media/${PLACED}/detail/1`,
    width: 1200,
    height: 600,
  });
  expect(fromEditor(editing)).toEqual(document);
});

test("a gallery keeps the order the author put its pictures in", () => {
  const second = "1a2b3c4d-5e6f-4071-8293-a4b5c6d7e8f9";
  const document: PostDocument = {
    version: POST_DOCUMENT_VERSION,
    content: [
      {
        type: "gallery",
        content: [
          { type: "galleryImage", mediaId: second, alt: "Second" },
          { type: "galleryImage", mediaId: PLACED, alt: "First" },
        ],
      },
    ],
  };
  expect(fromEditor(toEditor(document, HELD))).toEqual(document);
});

test("a picture with nothing said about it travels so Go can refuse it", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "image",
        attrs: {
          mediaId: PLACED,
          alt: "  ",
          caption: "",
          src: "/media/x/detail/1",
        },
      },
    ],
  });
  expect(carried.content).toEqual([
    { type: "image", mediaId: PLACED, alt: "" },
  ]);
});

test("the editor drops a picture that names no upload", () => {
  const carried = fromEditor({
    type: "doc",
    content: [
      {
        type: "image",
        attrs: { alt: "Hotlinked", src: "https://example.com/p.png" },
      },
    ],
  });
  expect(carried.content).toEqual([]);
});
