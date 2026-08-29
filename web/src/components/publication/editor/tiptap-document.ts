import type { JSONContent } from "@tiptap/react";
import {
  isPostAnchor,
  isPostCalloutKind,
  isPostLanguage,
  POST_DOCUMENT_VERSION,
  type PostBlock,
  type PostCell,
  type PostDocument,
  type PostItem,
  type PostMark,
  type PostRow,
  type PostSpan,
  type PostTask,
} from "@/lib/post-document";
import { isSafeAddress } from "@/lib/post-link";

/** The editor node each Illarin block is written as, and read back from. */
const EDITOR_NODES: Record<string, string> = {
  quote: "blockquote",
  divider: "horizontalRule",
};

const ILLARIN_BLOCKS: Record<string, string> = {
  blockquote: "quote",
  horizontalRule: "divider",
};

const MARKS = ["bold", "italic", "strike", "code"] as const;

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
        attrs: { level: block.level, anchor: block.anchor ?? null },
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
    case "taskList":
      return {
        type,
        content: block.content.map((task) => ({
          type: "taskItem",
          attrs: { checked: task.done },
          content: task.content.map(editorBlock),
        })),
      };
    case "quote":
      return { type, content: block.content.map(editorBlock) };
    case "callout":
      return {
        type,
        attrs: { kind: block.kind },
        content: block.content.map(editorBlock),
      };
    case "codeBlock":
      return {
        type,
        attrs: { language: block.language },
        content: [{ type: "text", text: block.source }],
      };
    case "table":
      return { type, content: block.content.map(editorRow) };
  }
}

function editorRow(row: PostRow): JSONContent {
  return {
    type: "tableRow",
    content: row.content.map((cell) => ({
      type: cell.heading ? "tableHeader" : "tableCell",
      content:
        cell.content.length > 0
          ? cell.content.map(editorBlock)
          : [{ type: "paragraph" }],
    })),
  };
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
      if (!written(spans)) return [];
      const anchor = node.attrs?.anchor;
      const kept = typeof anchor === "string" && isPostAnchor(anchor);
      return [
        {
          type: "heading",
          level: headingLevel(node.attrs?.level),
          ...(kept ? { anchor: anchor as string } : {}),
          content: spans,
        },
      ];
    }
    case "bulletList":
    case "orderedList": {
      const items = illarinItems(node.content);
      return items.length > 0 ? [{ type, content: items }] : [];
    }
    case "taskList": {
      const tasks = illarinTasks(node.content);
      return tasks.length > 0 ? [{ type, content: tasks }] : [];
    }
    case "quote": {
      const quoted = (node.content ?? []).flatMap(illarinBlock);
      return quoted.length > 0 ? [{ type: "quote", content: quoted }] : [];
    }
    case "callout": {
      const said = (node.content ?? []).flatMap(illarinBlock);
      const kind = node.attrs?.kind;
      return [
        {
          type: "callout",
          kind:
            typeof kind === "string" && isPostCalloutKind(kind) ? kind : "note",
          content:
            said.length > 0 ? said : [{ type: "paragraph", content: [] }],
        },
      ];
    }
    case "codeBlock": {
      const source = codeSource(node.content);
      if (!source.trim()) return [];
      const language = node.attrs?.language;
      return [
        {
          type: "codeBlock",
          language:
            typeof language === "string" && isPostLanguage(language)
              ? language
              : "plain",
          source,
        },
      ];
    }
    case "table": {
      const rows = illarinRows(node.content);
      return rows.length > 0 ? [{ type: "table", content: rows }] : [];
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

function illarinTasks(nodes: JSONContent[] | undefined): PostTask[] {
  return (nodes ?? [])
    .filter((node) => node.type === "taskItem")
    .map((node) => ({
      type: "taskItem" as const,
      done: node.attrs?.checked === true,
      content: (node.content ?? []).flatMap(illarinBlock),
    }))
    .filter((task) => task.content.length > 0);
}

// illarinRows squares the table off, because Go stores a rectangle and a
// pasted table can arrive with merged or missing cells.
function illarinRows(nodes: JSONContent[] | undefined): PostRow[] {
  const rows = (nodes ?? [])
    .filter((node) => node.type === "tableRow")
    .map((node) => (node.content ?? []).flatMap(illarinCells));
  const width = rows.reduce((widest, row) => Math.max(widest, row.length), 0);
  if (width === 0) return [];
  return rows.map((cells) => {
    const heading = cells.every((cell) => cell.heading);
    while (cells.length < width) {
      cells.push({
        type: "tableCell",
        ...(heading ? { heading: true as const } : {}),
        content: [],
      });
    }
    return { type: "tableRow" as const, content: cells };
  });
}

function illarinCells(node: JSONContent): PostCell[] {
  if (node.type !== "tableCell" && node.type !== "tableHeader") return [];
  const heading = node.type === "tableHeader";
  const content = (node.content ?? [])
    .filter((child) => child.type === "paragraph")
    .map((child) => ({
      type: "paragraph" as const,
      content: illarinSpans(child.content),
    }))
    .filter((child) => written(child.content));
  const cell: PostCell = {
    type: "tableCell",
    ...(heading ? { heading: true as const } : {}),
    content,
  };
  const across = Math.max(1, Number(node.attrs?.colspan) || 1);
  return [
    cell,
    ...Array.from({ length: across - 1 }, () => ({
      type: "tableCell" as const,
      ...(heading ? { heading: true as const } : {}),
      content: [],
    })),
  ];
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
  for (const name of MARKS) {
    if (marks?.some((mark) => mark.type === name)) carried.push({ type: name });
  }
  const link = marks?.find((mark) => mark.type === "link");
  const href = typeof link?.attrs?.href === "string" ? link.attrs.href : "";
  if (href && isSafeAddress(href)) carried.push({ type: "link", href });
  return carried;
}

function codeSource(nodes: JSONContent[] | undefined): string {
  return (nodes ?? [])
    .filter((node) => node.type === "text")
    .map((node) => node.text ?? "")
    .join("");
}

// written keeps a line the author has not typed into out of the working copy.
function written(spans: PostSpan[]): boolean {
  return spans.some((span) => span.text.trim().length > 0);
}

function headingLevel(value: unknown): number {
  return typeof value === "number" && value >= 2 && value <= 4 ? value : 2;
}
