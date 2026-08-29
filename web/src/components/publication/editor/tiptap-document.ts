import type { JSONContent } from "@tiptap/react";
import {
  POST_DOCUMENT_VERSION,
  type PostBlock,
  type PostDocument,
  type PostItem,
  type PostMark,
  type PostSpan,
} from "@/lib/post-document";

/** The editor node each Illarin block is written as, and read back from. */
const EDITOR_NODES: Record<string, string> = {
  quote: "blockquote",
  divider: "horizontalRule",
};

const ILLARIN_BLOCKS: Record<string, string> = {
  blockquote: "quote",
  horizontalRule: "divider",
};

export function toEditor(document: PostDocument): JSONContent {
  return { type: "doc", content: document.content.map(editorBlock) };
}

export function fromEditor(content: JSONContent): PostDocument {
  return {
    version: POST_DOCUMENT_VERSION,
    content: (content.content ?? []).flatMap(illarinBlock),
  };
}

function editorBlock(block: PostBlock): JSONContent {
  const type = EDITOR_NODES[block.type] ?? block.type;
  switch (block.type) {
    case "divider":
      return { type };
    case "heading":
      return {
        type,
        attrs: { level: block.level },
        content: block.content.map(editorSpan),
      };
    case "paragraph":
      return { type, content: block.content.map(editorSpan) };
    case "bulletList":
    case "orderedList":
      return {
        type,
        content: block.content.map((item) => ({
          type: "listItem",
          content: item.content.map(editorBlock),
        })),
      };
    case "quote":
      return { type, content: block.content.map(editorBlock) };
  }
}

function editorSpan(span: PostSpan): JSONContent {
  const marks = (span.marks ?? []).map((mark) =>
    mark.type === "link"
      ? { type: "link", attrs: { href: mark.href } }
      : { type: mark.type },
  );
  return marks.length > 0
    ? { type: "text", text: span.text, marks }
    : { type: "text", text: span.text };
}

function illarinBlock(node: JSONContent): PostBlock[] {
  const type = ILLARIN_BLOCKS[node.type ?? ""] ?? node.type;
  switch (type) {
    case "paragraph": {
      const spans = illarinSpans(node.content);
      return written(spans) ? [{ type: "paragraph", content: spans }] : [];
    }
    case "heading": {
      const spans = illarinSpans(node.content);
      return written(spans)
        ? [
            {
              type: "heading",
              level: headingLevel(node.attrs?.level),
              content: spans,
            },
          ]
        : [];
    }
    case "bulletList":
    case "orderedList": {
      const items = illarinItems(node.content);
      return items.length > 0 ? [{ type, content: items }] : [];
    }
    case "quote": {
      const quoted = (node.content ?? []).flatMap(illarinBlock);
      return quoted.length > 0 ? [{ type: "quote", content: quoted }] : [];
    }
    case "divider":
      return [{ type: "divider" }];
    default:
      return [];
  }
}

function illarinItems(nodes: JSONContent[] | undefined): PostItem[] {
  return (nodes ?? [])
    .filter((node) => node.type === "listItem")
    .map((node) => ({
      type: "listItem" as const,
      content: (node.content ?? []).flatMap(illarinBlock),
    }))
    .filter((item) => item.content.length > 0);
}

function illarinSpans(nodes: JSONContent[] | undefined): PostSpan[] {
  const spans: PostSpan[] = [];
  for (const node of nodes ?? []) {
    if (node.type !== "text" || !node.text) continue;
    const marks = illarinMarks(node.marks);
    spans.push(
      marks.length > 0
        ? { type: "text", text: node.text, marks }
        : { type: "text", text: node.text },
    );
  }
  return spans;
}

function illarinMarks(marks: JSONContent["marks"]): PostMark[] {
  const carried: PostMark[] = [];
  for (const name of ["bold", "italic", "code"] as const) {
    if (marks?.some((mark) => mark.type === name)) carried.push({ type: name });
  }
  const link = marks?.find((mark) => mark.type === "link");
  const href = typeof link?.attrs?.href === "string" ? link.attrs.href : "";
  if (href) carried.push({ type: "link", href });
  return carried;
}

// written keeps a line the author has not typed into out of the working copy.
function written(spans: PostSpan[]): boolean {
  return spans.some((span) => span.text.trim().length > 0);
}

function headingLevel(value: unknown): number {
  return typeof value === "number" && value >= 2 && value <= 4 ? value : 2;
}
