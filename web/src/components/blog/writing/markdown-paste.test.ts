import { expect, test } from "bun:test";
import { getSchema } from "@tiptap/core";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import StarterKit from "@tiptap/starter-kit";
import { Callout } from "./callout-node";
import { markdownPaste } from "./markdown-paste";
import { fromEditor } from "./tiptap-document";

const schema = getSchema([
  StarterKit.configure({ heading: { levels: [2, 3, 4] }, hardBreak: false }),
  TaskList,
  TaskItem.configure({ nested: true }),
  TableKit,
  Callout,
]);

test("pasted Markdown becomes formatted post content immediately", () => {
  const slice = markdownPaste(
    "## Heading\n\n- **Bold** and *italic*\n- [Link](https://example.com) with `code`\n\n> Quotation",
    schema,
  );
  expect(slice).not.toBeNull();
  expect(fromEditor({ content: slice?.content.toJSON() }).content).toEqual([
    { type: "heading", level: 2, content: [{ type: "text", text: "Heading" }] },
    {
      type: "bulletList",
      content: [
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [
                { type: "text", text: "Bold", marks: [{ type: "bold" }] },
                { type: "text", text: " and " },
                { type: "text", text: "italic", marks: [{ type: "italic" }] },
              ],
            },
          ],
        },
        {
          type: "listItem",
          content: [
            {
              type: "paragraph",
              content: [
                {
                  type: "text",
                  text: "Link",
                  marks: [{ type: "link", href: "https://example.com" }],
                },
                { type: "text", text: " with " },
                { type: "text", text: "code", marks: [{ type: "code" }] },
              ],
            },
          ],
        },
      ],
    },
    {
      type: "quote",
      content: [
        { type: "paragraph", content: [{ type: "text", text: "Quotation" }] },
      ],
    },
  ]);
});

test("tasks, tables, callouts and fenced code keep their structure", () => {
  const slice = markdownPaste(
    "- [x] Done\n- [ ] Waiting\n\n| Name |\n| --- |\n| Value |\n\n> [!TIP]\n> Helpful\n\n```javascript\nconst value = '**literal**';\n```",
    schema,
  );
  const body = fromEditor({ content: slice?.content.toJSON() });
  expect(body.content.map((block) => block.type)).toEqual([
    "taskList",
    "table",
    "callout",
    "codeBlock",
  ]);
  expect(body.content[0]).toMatchObject({
    content: [{ done: true }, { done: false }],
  });
  expect(body.content[1]).toMatchObject({
    content: [
      { content: [{ heading: true }] },
      { content: [{ content: [{ content: [{ text: "Value" }] }] }] },
    ],
  });
  expect(body.content[2]).toMatchObject({
    kind: "tip",
    content: [{ content: [{ text: "Helpful" }] }],
  });
  expect(body.content[3]).toEqual({
    type: "codeBlock",
    language: "javascript",
    source: "const value = '**literal**';",
  });
});

test("unsupported content falls back to the original paste without losing words", () => {
  for (const source of [
    "## Heading\n\n![Picture](https://example.com/image.png)",
    "**Keep** <script>literal</script>",
    "[Unsafe](javascript:alert(1))",
    "# Unsupported heading",
    "```unsupported\nsource\n```",
  ]) {
    expect(markdownPaste(source, schema)).toBeNull();
  }
});

test("ordinary text uses the editor's normal paste behavior", () => {
  expect(
    markdownPaste("First line\nSecond line\n\nAnother paragraph", schema),
  ).toBeNull();
  expect(markdownPaste("", schema)).toBeNull();
});
