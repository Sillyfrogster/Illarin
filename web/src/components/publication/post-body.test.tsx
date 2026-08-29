import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { PostMedia } from "@/lib/api/query";
import type { PostBlock, PostDocument } from "@/lib/post-document";
import { POST_DOCUMENT_VERSION } from "@/lib/post-document";
import { PostBody } from "./PostBody";

const PLACED = "6f2c1b40-9d38-4a7e-b512-0c8e37a41d59";
const SECOND = "1a2b3c4d-5e6f-4071-8293-a4b5c6d7e8f9";

const HELD: PostMedia[] = [PLACED, SECOND].map((id) => ({
  id,
  postId: "b8d0f1a2-3c45-4e67-89ab-cdef01234567",
  purpose: "document",
  url: `/media/${id}/detail/1`,
  thumbUrl: `/media/${id}/grid/1`,
  width: 1200,
  height: 600,
}));

function render(...content: PostBlock[]): string {
  const document: PostDocument = { version: POST_DOCUMENT_VERSION, content };
  return renderToStaticMarkup(<PostBody document={document} media={HELD} />);
}

test("a heading renders at its level and answers to its address", () => {
  const html = render({
    type: "heading",
    level: 3,
    anchor: "release-notes",
    content: [{ type: "text", text: "Release notes" }],
  });
  expect(html).toContain('<h3 id="release-notes">Release notes</h3>');
});

test("every mark reaches its own element", () => {
  const html = render({
    type: "paragraph",
    content: [
      { type: "text", text: "b", marks: [{ type: "bold" }] },
      { type: "text", text: "i", marks: [{ type: "italic" }] },
      { type: "text", text: "s", marks: [{ type: "strike" }] },
      { type: "text", text: "c", marks: [{ type: "code" }] },
    ],
  });
  expect(html).toContain("<strong>b</strong>");
  expect(html).toContain("<em>i</em>");
  expect(html).toContain("<s>s</s>");
  expect(html).toContain(">c</code>");
});

test("a link that leaves Illarin opens away and says so", () => {
  const html = render({
    type: "paragraph",
    content: [
      {
        type: "text",
        text: "Elsewhere",
        marks: [{ type: "link", href: "https://example.com/notes" }],
      },
    ],
  });
  expect(html).toContain('target="_blank"');
  expect(html).toContain('rel="noreferrer nofollow"');
  expect(html).toContain("opens in a new tab");
});

test("a link that stays on Illarin opens in the same tab", () => {
  const html = render({
    type: "paragraph",
    content: [
      {
        type: "text",
        text: "Here",
        marks: [{ type: "link", href: "http://localhost:8000/browse" }],
      },
    ],
  });
  expect(html).not.toContain('target="_blank"');
  expect(html).not.toContain("opens in a new tab");
});

test("a task list says what is done without offering a control", () => {
  const html = render({
    type: "taskList",
    content: [
      {
        type: "taskItem",
        done: true,
        content: [
          { type: "paragraph", content: [{ type: "text", text: "Shipped" }] },
        ],
      },
      {
        type: "taskItem",
        done: false,
        content: [
          { type: "paragraph", content: [{ type: "text", text: "Waiting" }] },
        ],
      },
    ],
  });
  expect(html).toContain('data-done="true"');
  expect(html).toContain('data-done="false"');
  expect(html).toContain("Done:");
  expect(html).toContain("To do:");
  expect(html).not.toContain("<input");
});

test("a callout names its kind", () => {
  const html = render({
    type: "callout",
    kind: "warning",
    content: [
      { type: "paragraph", content: [{ type: "text", text: "Careful." }] },
    ],
  });
  expect(html).toContain('data-kind="warning"');
  expect(html).toContain("Warning");
  expect(html).toContain('role="note"');
  expect(html).not.toContain("<aside");
});

test("a heading row becomes column headings and a heading column becomes row headings", () => {
  const cell = (text: string): PostBlock => ({
    type: "paragraph",
    content: [{ type: "text", text }],
  });
  const columns = render({
    type: "table",
    content: [
      {
        type: "tableRow",
        content: [
          { type: "tableCell", heading: true, content: [cell("Field")] },
          { type: "tableCell", heading: true, content: [cell("Type")] },
        ],
      },
      {
        type: "tableRow",
        content: [
          { type: "tableCell", content: [cell("title")] },
          { type: "tableCell", content: [cell("string")] },
        ],
      },
    ],
  });
  expect(columns).toContain("<thead>");
  expect(columns).toContain('scope="col"');
  expect(columns).not.toContain('scope="row"');

  const rowHeadings = render({
    type: "table",
    content: [
      {
        type: "tableRow",
        content: [
          { type: "tableCell", heading: true, content: [cell("Draft")] },
          { type: "tableCell", content: [cell("no")] },
        ],
      },
      {
        type: "tableRow",
        content: [
          { type: "tableCell", heading: true, content: [cell("Published")] },
          { type: "tableCell", content: [cell("yes")] },
        ],
      },
    ],
  });
  expect(rowHeadings).not.toContain("<thead>");
  expect(rowHeadings).toContain('scope="row"');
});

test("code keeps its source and is highlighted from it", () => {
  const html = render({
    type: "codeBlock",
    language: "go",
    source: 'func main() {\n\t// note\n\tprint("hi")\n}',
  });
  expect(html).toContain("Go</span>");
  expect(html).toContain("<span>// note</span>");
  expect(html).toContain("<span>func</span>");
  expect(html).toContain("&quot;hi&quot;");
});

test("plain code is not labelled and carries no highlighting classes", () => {
  const html = render({
    type: "codeBlock",
    language: "plain",
    source: "illarin publish",
  });
  expect(html).toContain("illarin publish");
  expect(html).not.toContain("Plain text</span>");
});

test("authored text never becomes markup", () => {
  const html = render(
    {
      type: "paragraph",
      content: [{ type: "text", text: '<script>alert("x")</script>' }],
    },
    {
      type: "codeBlock",
      language: "html",
      source: "<img onerror=alert(1)>",
    },
  );
  expect(html).not.toContain("<script>");
  expect(html).not.toContain("onerror=alert");
  expect(html).toContain("&lt;script&gt;");
});

test("an address outside the safe schemes never becomes a link", () => {
  const html = render({
    type: "paragraph",
    content: [
      {
        type: "text",
        text: "Run it",
        marks: [{ type: "link", href: "javascript:alert(1)" }],
      },
    ],
  });
  expect(html).toContain("Run it");
  expect(html).not.toContain("<a");
  expect(html).not.toContain("javascript:");
});

test("a picture renders as a figure with its description and caption", () => {
  const html = render({
    type: "image",
    mediaId: PLACED,
    alt: "The workspace with one draft in it",
    caption: "The workspace after the first draft is started.",
  });
  expect(html).toContain(`src="/media/${PLACED}/detail/1"`);
  expect(html).toContain('alt="The workspace with one draft in it"');
  expect(html).toContain(
    "<figcaption>The workspace after the first draft is started.</figcaption>",
  );
});

test("a gallery renders its pictures in the order the author put them in", () => {
  const html = render({
    type: "gallery",
    content: [
      { type: "galleryImage", mediaId: SECOND, alt: "Second" },
      { type: "galleryImage", mediaId: PLACED, alt: "First" },
    ],
  });
  expect(html.indexOf(SECOND)).toBeLessThan(html.indexOf(PLACED));
  expect(html).toContain(`src="/media/${SECOND}/grid/1"`);
});

test("a picture whose bytes are gone renders nothing rather than a broken frame", () => {
  const html = render({
    type: "image",
    mediaId: "00000000-0000-4000-8000-000000000000",
    alt: "A picture that was collected",
  });
  expect(html).not.toContain("<img");
});
