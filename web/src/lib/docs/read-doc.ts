import type {
  BlockContent,
  Blockquote,
  DefinitionContent,
  PhrasingContent,
  RootContent,
  Table,
} from "mdast";
import { fromMarkdown } from "mdast-util-from-markdown";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { gfm } from "micromark-extension-gfm";

export type DocInline =
  | { kind: "text"; text: string }
  | { kind: "code"; text: string }
  | { kind: "strong"; children: DocInline[] }
  | { kind: "emphasis"; children: DocInline[] }
  | { kind: "link"; href: string; children: DocInline[] };

export type DocTone = "note" | "tip" | "important" | "warning";

export type DocBlock =
  | { kind: "paragraph"; children: DocInline[] }
  | { kind: "endpoint"; method: string; path: string }
  | { kind: "heading"; level: 2 | 3; anchor: string; children: DocInline[] }
  | { kind: "list"; ordered: boolean; items: DocBlock[][] }
  | { kind: "code"; language: string; label: string; source: string }
  | { kind: "table"; head: DocInline[][]; rows: DocInline[][][] }
  | { kind: "callout"; tone: DocTone; children: DocBlock[] }
  | { kind: "quote"; children: DocBlock[] }
  | { kind: "divider" };

export type DocContentsEntry = { anchor: string; label: string; level: 2 | 3 };

export type Doc = {
  title: string;
  lede: string;
  blocks: DocBlock[];
  contents: DocContentsEntry[];
};

const TONES: DocTone[] = ["note", "tip", "important", "warning"];

const ALERT = /^\[!([A-Z]+)\]\s*/;

const ENDPOINT = /^(GET|POST|PUT|PATCH|DELETE) (\/\S+)$/;

type Child = RootContent | BlockContent | DefinitionContent;

/** Reads one documentation page written in constrained Markdown. */
export function readDoc(source: string): Doc {
  const tree = fromMarkdown(source, {
    extensions: [gfm()],
    mdastExtensions: [gfmFromMarkdown()],
  });
  const [first, second, ...rest] = tree.children;
  if (!first || first.type !== "heading" || first.depth !== 1) {
    throw new Error("A doc starts with a level-one heading.");
  }
  if (!second || second.type !== "paragraph") {
    throw new Error("A doc opens with one paragraph beneath its title.");
  }
  const anchors = new Map<string, number>();
  const blocks = rest.map((node) => readBlock(node, anchors));
  return {
    title: plainText(first.children),
    lede: plainText(second.children),
    blocks,
    contents: blocks.flatMap((block) =>
      block.kind === "heading"
        ? [
            {
              anchor: block.anchor,
              label: inlineText(block.children),
              level: block.level,
            },
          ]
        : [],
    ),
  };
}

function readBlock(node: Child, anchors: Map<string, number>): DocBlock {
  switch (node.type) {
    case "paragraph": {
      const children = readInlines(node.children);
      const only = children.length === 1 ? children[0] : null;
      const call = only?.kind === "code" ? ENDPOINT.exec(only.text) : null;
      if (call) return { kind: "endpoint", method: call[1], path: call[2] };
      return { kind: "paragraph", children };
    }
    case "heading": {
      if (node.depth !== 2 && node.depth !== 3) {
        throw new Error("A doc's body headings are level two or three.");
      }
      const children = readInlines(node.children);
      return {
        kind: "heading",
        level: node.depth,
        anchor: uniqueAnchor(plainText(node.children), anchors),
        children,
      };
    }
    case "list":
      return {
        kind: "list",
        ordered: node.ordered === true,
        items: node.children.map((item) =>
          item.children.map((child) => readBlock(child, anchors)),
        ),
      };
    case "code":
      return {
        kind: "code",
        language: node.lang ?? "plain",
        label: node.meta?.trim() || (node.lang ?? "plain"),
        source: node.value,
      };
    case "table":
      return readTable(node);
    case "blockquote":
      return readQuote(node, anchors);
    case "thematicBreak":
      return { kind: "divider" };
    default:
      throw new Error(`Docs do not carry ${node.type}.`);
  }
}

function readTable(node: Table): DocBlock {
  const [head, ...rows] = node.children.map((row) =>
    row.children.map((cell) => readInlines(cell.children)),
  );
  if (!head) throw new Error("A table needs a heading row.");
  return { kind: "table", head, rows };
}

function readQuote(node: Blockquote, anchors: Map<string, number>): DocBlock {
  const [first, ...rest] = node.children;
  const lead = first?.type === "paragraph" ? first.children[0] : undefined;
  const marker = lead?.type === "text" ? ALERT.exec(lead.value) : null;
  if (!marker || first?.type !== "paragraph" || lead?.type !== "text") {
    return {
      kind: "quote",
      children: node.children.map((child) => readBlock(child, anchors)),
    };
  }
  const tone = marker[1].toLowerCase();
  if (!isTone(tone)) {
    throw new Error(`A callout is one of: ${TONES.join(", ")}.`);
  }
  const opening: PhrasingContent[] = [
    { type: "text", value: lead.value.slice(marker[0].length) },
    ...first.children.slice(1),
  ];
  const children = [
    ...(plainText(opening).length > 0
      ? [{ kind: "paragraph" as const, children: readInlines(opening) }]
      : []),
    ...rest.map((child) => readBlock(child, anchors)),
  ];
  return { kind: "callout", tone, children };
}

function readInlines(nodes: PhrasingContent[]): DocInline[] {
  const out: DocInline[] = [];
  for (const node of nodes) {
    switch (node.type) {
      case "text":
        pushText(out, node.value);
        break;
      case "break":
        pushText(out, " ");
        break;
      case "inlineCode":
        out.push({ kind: "code", text: node.value });
        break;
      case "strong":
        out.push({ kind: "strong", children: readInlines(node.children) });
        break;
      case "emphasis":
        out.push({ kind: "emphasis", children: readInlines(node.children) });
        break;
      case "link":
        out.push({
          kind: "link",
          href: node.url,
          children: readInlines(node.children),
        });
        break;
      default:
        throw new Error(`Docs do not carry ${node.type}.`);
    }
  }
  return out;
}

function pushText(out: DocInline[], text: string) {
  const last = out.at(-1);
  if (last?.kind === "text") {
    out[out.length - 1] = { kind: "text", text: last.text + text };
    return;
  }
  out.push({ kind: "text", text });
}

function plainText(nodes: PhrasingContent[]): string {
  return nodes
    .map((node) => {
      if (node.type === "text" || node.type === "inlineCode") return node.value;
      if (node.type === "break") return " ";
      if ("children" in node) return plainText(node.children);
      return "";
    })
    .join("")
    .trim();
}

export function inlineText(nodes: DocInline[]): string {
  return nodes
    .map((node) => ("text" in node ? node.text : inlineText(node.children)))
    .join("");
}

function uniqueAnchor(label: string, anchors: Map<string, number>): string {
  const stem = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
  const seen = (anchors.get(stem) ?? 0) + 1;
  anchors.set(stem, seen);
  return seen === 1 ? stem : `${stem}-${seen}`;
}

function isTone(tone: string): tone is DocTone {
  return (TONES as string[]).includes(tone);
}
