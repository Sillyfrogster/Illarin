import type { JSONContent } from "@tiptap/core";
import { type Schema, Slice } from "@tiptap/pm/model";
import type { Nodes } from "mdast";
import { fromMarkdown } from "mdast-util-from-markdown";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { gfm } from "micromark-extension-gfm";
import { isPostCalloutKind, isPostLanguage } from "@/lib/post-body";
import { isSafeAddress } from "@/lib/post-link";

export function markdownPaste(source: string, schema: Schema): Slice | null {
  const tree = fromMarkdown(source, {
    extensions: [gfm()],
    mdastExtensions: [gfmFromMarkdown()],
  });
  let supported = true;
  let formatted = false;

  function read(node: Nodes): JSONContent[] {
    const content = () =>
      "children" in node ? node.children.flatMap(read) : [];
    const block = (
      type: string,
      attrs?: JSONContent["attrs"],
    ): JSONContent[] => [
      { type, ...(attrs ? { attrs } : {}), content: content() },
    ];
    const mark = (type: string, attrs?: JSONContent["attrs"]): JSONContent[] =>
      content().map((child) => ({
        ...child,
        marks: [...(child.marks ?? []), { type, ...(attrs ? { attrs } : {}) }],
      }));

    if (node.type !== "text" && node.type !== "paragraph") formatted = true;
    switch (node.type) {
      case "text":
        return node.value ? [{ type: "text", text: node.value }] : [];
      case "paragraph":
        return block("paragraph");
      case "heading":
        if (node.depth < 2 || node.depth > 4) break;
        return block("heading", { level: node.depth });
      case "strong":
        return mark("bold");
      case "emphasis":
        return mark("italic");
      case "delete":
        return mark("strike");
      case "inlineCode":
        return node.value
          ? [{ type: "text", text: node.value, marks: [{ type: "code" }] }]
          : [];
      case "link":
        if (!isSafeAddress(node.url)) break;
        return mark("link", { href: node.url });
      case "list": {
        const tasks = node.children.some((item) => item.checked != null);
        if (tasks && node.children.some((item) => item.checked == null)) break;
        return block(
          tasks ? "taskList" : node.ordered ? "orderedList" : "bulletList",
        );
      }
      case "listItem":
        return block(node.checked == null ? "listItem" : "taskItem", {
          checked: node.checked === true,
        });
      case "blockquote": {
        const first = node.children[0];
        const text = first?.type === "paragraph" ? first.children[0] : null;
        const marker =
          text?.type === "text" && /^\[!(\w+)\](?:\n|$)/.exec(text.value);
        if (!marker) return block("blockquote");
        const kind = marker[1].toLowerCase();
        if (!isPostCalloutKind(kind)) break;
        text.value = text.value.slice(marker[0].length);
        const children = content().filter((child) => child.content?.length);
        return [{ type: "callout", attrs: { kind }, content: children }];
      }
      case "code":
        if (node.lang && !isPostLanguage(node.lang)) break;
        return [
          {
            type: "codeBlock",
            attrs: { language: node.lang || "plain" },
            content: node.value ? [{ type: "text", text: node.value }] : [],
          },
        ];
      case "thematicBreak":
        return [{ type: "horizontalRule" }];
      case "table":
        return [
          {
            type: "table",
            content: node.children.map((row, index) => ({
              type: "tableRow",
              content: row.children.map((cell) => ({
                type: index === 0 ? "tableHeader" : "tableCell",
                content: [
                  { type: "paragraph", content: cell.children.flatMap(read) },
                ],
              })),
            })),
          },
        ];
    }
    supported = false;
    return [];
  }

  const content = tree.children.flatMap(read);
  if (!supported || !formatted) return null;
  const document = schema.nodeFromJSON({ type: "doc", content });
  try {
    document.check();
  } catch {
    return null;
  }
  return Slice.maxOpen(document.content);
}
