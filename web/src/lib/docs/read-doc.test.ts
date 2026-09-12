import { expect, test } from "bun:test";
import { readDoc } from "./read-doc";

const PAGE = `# Webhooks

Every public transition is sent as a signed request.

## Events

Three events exist.

### Published

The first one.

### Published

A second heading with the same words.

## Verify the signature

\`\`\`http POST /v1/publication/posts
POST /api/v1/publication/posts HTTP/1.1
Authorization: Bearer ip1.example

{"title": "Hello"}
\`\`\`

\`\`\`json Post
{"id": "1"}
\`\`\`

| Header | Meaning |
| --- | --- |
| \`webhook-id\` | The delivery |
| \`webhook-timestamp\` | Unix seconds |

> [!WARNING]
> Compare in constant time.

> A plain quotation.

- One with **bold** and [a link](/developers/publication)
- Two with \`code\`

1. First
2. Second

---
`;

test("the first heading is the title and the paragraph after it the lede", () => {
  const doc = readDoc(PAGE);
  expect(doc.title).toBe("Webhooks");
  expect(doc.lede).toBe("Every public transition is sent as a signed request.");
  expect(doc.blocks[0]).toEqual({
    kind: "heading",
    level: 2,
    anchor: "events",
    children: [{ kind: "text", text: "Events" }],
  });
});

test("headings get anchors, repeats are numbered and the contents list them", () => {
  const doc = readDoc(PAGE);
  expect(doc.contents).toEqual([
    { anchor: "events", label: "Events", level: 2 },
    { anchor: "published", label: "Published", level: 3 },
    { anchor: "published-2", label: "Published", level: 3 },
    { anchor: "verify-the-signature", label: "Verify the signature", level: 2 },
  ]);
});

test("a code fence keeps its language, its label and its source", () => {
  const doc = readDoc(PAGE);
  const fences = doc.blocks.filter((block) => block.kind === "code");
  expect(fences).toEqual([
    {
      kind: "code",
      language: "http",
      label: "POST /v1/publication/posts",
      source:
        'POST /api/v1/publication/posts HTTP/1.1\nAuthorization: Bearer ip1.example\n\n{"title": "Hello"}',
    },
    { kind: "code", language: "json", label: "Post", source: '{"id": "1"}' },
  ]);
});

test("a table splits into its heading row and its body rows", () => {
  const doc = readDoc(PAGE);
  const table = doc.blocks.find((block) => block.kind === "table");
  expect(table).toEqual({
    kind: "table",
    head: [
      [{ kind: "text", text: "Header" }],
      [{ kind: "text", text: "Meaning" }],
    ],
    rows: [
      [
        [{ kind: "code", text: "webhook-id" }],
        [{ kind: "text", text: "The delivery" }],
      ],
      [
        [{ kind: "code", text: "webhook-timestamp" }],
        [{ kind: "text", text: "Unix seconds" }],
      ],
    ],
  });
});

test("an alert quotation is a callout and a plain one stays a quotation", () => {
  const doc = readDoc(PAGE);
  const asides = doc.blocks.filter(
    (block) => block.kind === "callout" || block.kind === "quote",
  );
  expect(asides).toEqual([
    {
      kind: "callout",
      tone: "warning",
      children: [
        {
          kind: "paragraph",
          children: [{ kind: "text", text: "Compare in constant time." }],
        },
      ],
    },
    {
      kind: "quote",
      children: [
        {
          kind: "paragraph",
          children: [{ kind: "text", text: "A plain quotation." }],
        },
      ],
    },
  ]);
});

test("lists carry their items and inline marks", () => {
  const doc = readDoc(PAGE);
  const lists = doc.blocks.filter((block) => block.kind === "list");
  expect(lists).toEqual([
    {
      kind: "list",
      ordered: false,
      items: [
        [
          {
            kind: "paragraph",
            children: [
              { kind: "text", text: "One with " },
              { kind: "strong", children: [{ kind: "text", text: "bold" }] },
              { kind: "text", text: " and " },
              {
                kind: "link",
                href: "/developers/publication",
                children: [{ kind: "text", text: "a link" }],
              },
            ],
          },
        ],
        [
          {
            kind: "paragraph",
            children: [
              { kind: "text", text: "Two with " },
              { kind: "code", text: "code" },
            ],
          },
        ],
      ],
    },
    {
      kind: "list",
      ordered: true,
      items: [
        [{ kind: "paragraph", children: [{ kind: "text", text: "First" }] }],
        [{ kind: "paragraph", children: [{ kind: "text", text: "Second" }] }],
      ],
    },
  ]);
  expect(doc.blocks.at(-1)).toEqual({ kind: "divider" });
});

test("a paragraph holding only a method and path is an endpoint line", () => {
  const doc = readDoc(
    "# T\n\nLede.\n\n`PUT /api/v1/publication/posts/{id}`\n\n`PUT` alone is prose.",
  );
  expect(doc.blocks).toEqual([
    { kind: "endpoint", method: "PUT", path: "/api/v1/publication/posts/{id}" },
    {
      kind: "paragraph",
      children: [
        { kind: "code", text: "PUT" },
        { kind: "text", text: " alone is prose." },
      ],
    },
  ]);
});

test("a page without a title or lede is refused", () => {
  expect(() => readDoc("## No title\n\nWords.")).toThrow(
    "A doc starts with a level-one heading.",
  );
  expect(() => readDoc("# Title\n\n## Straight in")).toThrow(
    "A doc opens with one paragraph beneath its title.",
  );
});

test("html and pictures are refused rather than dropped", () => {
  expect(() => readDoc("# T\n\nLede.\n\n<div>no</div>")).toThrow(
    "Docs do not carry html.",
  );
  expect(() => readDoc("# T\n\nLede.\n\n![alt](/x.png)")).toThrow(
    "Docs do not carry image.",
  );
});
