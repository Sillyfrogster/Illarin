import type { PhrasingContent, RootContent } from "mdast";
import { fromMarkdown } from "mdast-util-from-markdown";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { gfm } from "micromark-extension-gfm";

export type RichInline =
  | { kind: "text"; text: string }
  | { kind: "break" }
  | { kind: "code"; text: string }
  | { kind: "emphasis"; children: RichInline[] }
  | { kind: "strong"; children: RichInline[] }
  | { kind: "delete"; children: RichInline[] }
  | { kind: "link"; href: string; children: RichInline[] };

export type RichAlign = "center" | "right";

export type RichBlock =
  | { kind: "paragraph"; children: RichInline[]; align?: RichAlign }
  | {
      kind: "heading";
      depth: number;
      children: RichInline[];
      align?: RichAlign;
    }
  | { kind: "quote"; children: RichBlock[] }
  | {
      kind: "callout";
      tone: "accent" | "stop";
      title: string;
      children: RichBlock[];
    }
  | {
      kind: "list";
      ordered: boolean;
      start: number;
      items: RichBlock[][];
      checked?: (boolean | null)[];
    }
  | { kind: "code"; text: string }
  | { kind: "rule" }
  | {
      kind: "table";
      head: RichInline[][] | null;
      rows: RichInline[][][];
    };

export type RichText = {
  blocks: RichBlock[];
  formattingRemoved: boolean;
};

export function readRichText(source: string): RichText {
  const stripped = stripHtml(source);
  const removed = { formatting: stripped.removed };
  const blocks = readRentry(stripped.text, removed);
  const shallowest = shallowestHeading(blocks) ?? 1;
  return {
    blocks: shallowest > 1 ? raiseHeadings(blocks, shallowest - 1) : blocks,
    formattingRemoved: removed.formatting,
  };
}

function shallowestHeading(blocks: RichBlock[]): number | null {
  const depths: number[] = [];
  for (const block of blocks) {
    if (block.kind === "heading") depths.push(block.depth);
    if (block.kind === "quote" || block.kind === "callout") {
      const inner = shallowestHeading(block.children);
      if (inner !== null) depths.push(inner);
    }
    if (block.kind === "list") {
      for (const item of block.items) {
        const inner = shallowestHeading(item);
        if (inner !== null) depths.push(inner);
      }
    }
  }
  return depths.length === 0 ? null : Math.min(...depths);
}

function raiseHeadings(blocks: RichBlock[], by: number): RichBlock[] {
  return blocks.map((block) => {
    if (block.kind === "heading") return { ...block, depth: block.depth - by };
    if (block.kind === "quote" || block.kind === "callout") {
      return { ...block, children: raiseHeadings(block.children, by) };
    }
    if (block.kind === "list") {
      return {
        ...block,
        items: block.items.map((item) => raiseHeadings(item, by)),
      };
    }
    return block;
  });
}

export function formattingWasRemoved(texts: readonly string[]): boolean {
  return texts.some((text) => readRichText(text).formattingRemoved);
}

export function richTextsOf(element: {
  type: string;
  display?: string;
  content: unknown;
}): string[] {
  if (!element.content || typeof element.content !== "object") return [];
  const content = element.content as Record<string, unknown>;
  const verbatim = element.display === "verbatim";
  switch (element.type) {
    case "prose":
      return verbatim ? [] : texts([content], "text");
    case "text_set":
      return verbatim ? [] : texts(items(content.texts), "text");
    case "dialogue_sample":
      return texts(items(content.turns), "text");
    case "entry_table":
      return texts(items(content.entries), "text");
    case "field_list":
      return texts(items(content.fields), "value");
    case "link_list":
      return texts(items(content.links), "note");
    case "prompt_list":
      return texts(items(content.fragments), "text");
    case "variable_schema":
      return texts(items(content.variables), "description");
    case "record_list":
      return texts(items(content.records), "lumiaDefinition");
    default:
      return [];
  }
}

function items(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? (value as Record<string, unknown>[]) : [];
}

function texts(entries: Record<string, unknown>[], key: string): string[] {
  const found: string[] = [];
  for (const entry of entries) {
    const value = entry?.[key];
    if (typeof value === "string" && value.trim() !== "") found.push(value);
  }
  return found;
}

const DISABLED = {
  disable: { null: ["codeIndented", "htmlFlow", "htmlText"] },
};

type Removed = { formatting: boolean };

function readMarkdown(source: string, removed: Removed): RichBlock[] {
  const tree = fromMarkdown(source, {
    extensions: [DISABLED, gfm()],
    mdastExtensions: [gfmFromMarkdown()],
  });
  return readBlocks(tree.children, removed);
}

const CALLOUT = /^!!!\s*(\w+)?(?:[ \t]+(.*))?$/;
const CALLOUT_BODY = /^(?: {4}|\t)/;
const CONTENTS = /^\s*\[TOC\d?\]\s*$/i;
const COLOUR = /%[#\w]+%([\s\S]*?)%%/g;
const STOP_TONES = new Set(["warning", "danger", "caution", "error"]);

/** Reads rentry's callouts, contents markers and colour marks around the Markdown, outside fenced code. */
function readRentry(source: string, removed: Removed): RichBlock[] {
  const blocks: RichBlock[] = [];
  let pending: string[] = [];
  const flush = () => {
    if (pending.length > 0)
      blocks.push(...readMarkdown(pending.join("\n"), removed));
    pending = [];
  };
  for (const run of fencedRuns(source)) {
    if (run.fenced) {
      pending.push(run.text);
      continue;
    }
    const lines = run.text.split("\n");
    for (let at = 0; at < lines.length; at++) {
      const line = lines[at];
      const callout = CALLOUT.exec(line);
      if (!callout) {
        if (!CONTENTS.test(line)) pending.push(line.replace(COLOUR, "$1"));
        continue;
      }
      flush();
      const body: string[] = [];
      while (at + 1 < lines.length && CALLOUT_BODY.test(lines[at + 1])) {
        body.push(lines[++at].replace(CALLOUT_BODY, ""));
      }
      const rest = (callout[2] ?? "").trim();
      const words = body.length > 0 ? body.join("\n") : rest;
      blocks.push({
        kind: "callout",
        tone: STOP_TONES.has((callout[1] ?? "").toLowerCase())
          ? "stop"
          : "accent",
        title: body.length > 0 ? rest.replace(COLOUR, "$1") : "",
        children: readRentry(words, removed),
      });
    }
  }
  flush();
  return blocks.map(aligned);
}

/** Rentry centres a block between -> and <-, and right-aligns it between -> and ->. */
function aligned(block: RichBlock): RichBlock {
  if (block.kind !== "paragraph" && block.kind !== "heading") return block;
  const first = edgeText(block.children, "first");
  const last = edgeText(block.children, "last");
  if (!first || !last || !first.text.trimStart().startsWith("->")) return block;
  const ending = last.text.trimEnd();
  const align = ending.endsWith("<-")
    ? "center"
    : ending.endsWith("->")
      ? "right"
      : null;
  if (!align || (first === last && ending.length < 4)) return block;
  first.text = first.text.trimStart().slice(2).trimStart();
  last.text = last.text.trimEnd().slice(0, -2).trimEnd();
  return { ...block, align };
}

function edgeText(
  nodes: RichInline[],
  edge: "first" | "last",
): { kind: "text"; text: string } | null {
  const node = edge === "first" ? nodes[0] : nodes.at(-1);
  if (!node) return null;
  if (node.kind === "text") return node;
  if (
    node.kind === "emphasis" ||
    node.kind === "strong" ||
    node.kind === "delete"
  ) {
    return edgeText(node.children, edge);
  }
  return null;
}

function readBlocks(nodes: RootContent[], removed: Removed): RichBlock[] {
  const blocks: RichBlock[] = [];
  for (const node of nodes) {
    switch (node.type) {
      case "paragraph": {
        const children = readInline(node.children, removed);
        if (!isBlank(children)) blocks.push({ kind: "paragraph", children });
        break;
      }
      case "heading": {
        const children = readInline(node.children, removed);
        if (!isBlank(children)) {
          blocks.push({ kind: "heading", depth: node.depth, children });
        }
        break;
      }
      case "blockquote": {
        const children = readBlocks(node.children, removed);
        if (children.length > 0) blocks.push({ kind: "quote", children });
        break;
      }
      case "list": {
        const held = node.children
          .map((item) => ({
            blocks: readBlocks(item.children, removed),
            checked: item.checked ?? null,
          }))
          .filter((item) => item.blocks.length > 0);
        if (held.length > 0) {
          const checked = held.map((item) => item.checked);
          blocks.push({
            kind: "list",
            ordered: node.ordered ?? false,
            start: node.start ?? 1,
            items: held.map((item) => item.blocks),
            ...(checked.some((one) => one !== null) ? { checked } : {}),
          });
        }
        break;
      }
      case "thematicBreak":
        blocks.push({ kind: "rule" });
        break;
      case "code": {
        if (node.value.trim() !== "") {
          blocks.push({ kind: "code", text: node.value });
        }
        break;
      }
      case "table": {
        const [head = [], ...rows] = node.children.map((row) =>
          row.children.map((cell) => readInline(cell.children, removed)),
        );
        blocks.push({
          kind: "table",
          head: head.every(isBlank) ? null : head,
          rows,
        });
        break;
      }
      case "definition":
      case "footnoteDefinition":
        break;
      default: {
        removed.formatting = true;
        const words = flatten(node);
        if (words.trim() !== "") {
          blocks.push({ kind: "paragraph", children: readLines(words) });
        }
      }
    }
  }
  return blocks;
}

function readInline(nodes: PhrasingContent[], removed: Removed): RichInline[] {
  const children: RichInline[] = [];
  for (const node of nodes) {
    switch (node.type) {
      case "text":
        children.push(...readLines(node.value));
        break;
      case "emphasis":
        children.push({
          kind: "emphasis",
          children: readInline(node.children, removed),
        });
        break;
      case "strong":
        children.push({
          kind: "strong",
          children: readInline(node.children, removed),
        });
        break;
      case "delete":
        children.push({
          kind: "delete",
          children: readInline(node.children, removed),
        });
        break;
      case "inlineCode":
        children.push({ kind: "code", text: node.value });
        break;
      case "break":
        children.push({ kind: "break" });
        break;
      case "link": {
        const href = readHref(node.url);
        const inner = readInline(node.children, removed);
        if (href) {
          children.push({ kind: "link", href, children: inner });
        } else {
          removed.formatting = true;
          children.push(...inner);
        }
        break;
      }
      case "image":
      case "imageReference":
        removed.formatting = true;
        children.push(...readLines(node.alt ?? ""));
        break;
      case "linkReference":
        removed.formatting = true;
        children.push(...readInline(node.children, removed));
        break;
      default:
        removed.formatting = true;
        children.push(...readLines(flatten(node)));
    }
  }
  return children;
}

function readLines(value: string): RichInline[] {
  const children: RichInline[] = [];
  value.split("\n").forEach((part, index) => {
    if (index > 0) children.push({ kind: "break" });
    if (part !== "") children.push({ kind: "text", text: part });
  });
  return children;
}

function isBlank(children: RichInline[]): boolean {
  return children.every((child) => {
    if (child.kind === "break") return true;
    if (child.kind === "text" || child.kind === "code") {
      return child.text.trim() === "";
    }
    if (child.kind === "link") return false;
    return isBlank(child.children);
  });
}

function flatten(node: unknown): string {
  if (!node || typeof node !== "object") return "";
  const branch = node as { value?: unknown; children?: unknown };
  if (typeof branch.value === "string") return branch.value;
  if (!Array.isArray(branch.children)) return "";
  return branch.children.map(flatten).join("");
}

const FOLLOWABLE_SCHEMES = new Set(["http", "https", "mailto"]);

function readHref(url: string): string | null {
  const cleaned = url.replace(/[\s\p{Cc}]/gu, "");
  if (cleaned === "") return null;
  const scheme = /^([a-zA-Z][a-zA-Z0-9+.-]*):/.exec(cleaned);
  if (scheme) {
    return FOLLOWABLE_SCHEMES.has(scheme[1].toLowerCase()) ? cleaned : null;
  }
  return cleaned.startsWith("/") && !cleaned.startsWith("//") ? cleaned : null;
}

const DISCARDED = /<(script|style|svg)\b[^>]*>[\s\S]*?(?:<\/\1\s*>|$)/gi;

const COMMENT_OR_TAG =
  /<!--[\s\S]*?-->|<\/?([a-zA-Z][a-zA-Z0-9-]*)(?:\s(?:"[^"]*"|'[^']*'|[^'">])*)?\/?>/g;

/** Reduces HTML to its words everywhere except inside a fenced code block. */
function stripHtml(source: string): { text: string; removed: boolean } {
  let removed = false;
  const runs = fencedRuns(source).map((run) => {
    if (run.fenced) return run.text;
    const reduced = reduceHtml(run.text);
    removed ||= reduced.removed;
    return reduced.text;
  });
  if (!removed) return { text: source, removed: false };
  return { text: runs.join("\n").trim(), removed };
}

function reduceHtml(source: string): { text: string; removed: boolean } {
  let removed = false;
  let text = source.replace(DISCARDED, () => {
    removed = true;
    return "";
  });
  text = text.replace(COMMENT_OR_TAG, (match: string, name?: string) => {
    if (name === undefined) {
      removed = true;
      return "";
    }
    const tag = name.toLowerCase();
    if (!HTML_ELEMENTS.has(tag)) return match;
    removed = true;
    return BLOCK_ELEMENTS.has(tag) ? "\n" : "";
  });
  if (!removed) return { text: source, removed: false };
  const flat = text
    .split("\n")
    .map((line) => line.trim())
    .join("\n");
  return { text: flat.replace(/\n{3,}/g, "\n\n"), removed };
}

const FENCE = /^ {0,3}(`{3,}|~{3,})/;

/** Splits writing into runs of lines inside and outside fenced code blocks. */
function fencedRuns(source: string): { text: string; fenced: boolean }[] {
  const runs: { lines: string[]; fenced: boolean }[] = [];
  let open: string | null = null;
  for (const line of source.split("\n")) {
    const fence = FENCE.exec(line)?.[1];
    const fenced = open !== null || fence !== undefined;
    const last = runs.at(-1);
    if (last?.fenced === fenced) last.lines.push(line);
    else runs.push({ lines: [line], fenced });
    if (open === null) {
      open = fence ?? null;
    } else if (fence?.[0] === open[0] && fence.length >= open.length) {
      open = line.trim() === fence ? null : open;
    }
  }
  return runs.map((run) => ({
    text: run.lines.join("\n"),
    fenced: run.fenced,
  }));
}

const HTML_ELEMENTS = new Set(
  `a abbr acronym address applet area article aside audio b base basefont bdi
   bdo bgsound big blink blockquote body br button canvas caption center cite
   code col colgroup data datalist dd del details dfn dialog dir div dl dt em
   embed fieldset figcaption figure font footer form frame frameset h1 h2 h3 h4
   h5 h6 head header hgroup hr html i iframe image img input ins kbd keygen
   label legend li link listing main map mark marquee math menu menuitem meta
   meter nav nobr noembed noframes noscript object ol optgroup option output p
   param picture plaintext pre progress q rb rp rt rtc ruby s samp script search
   section select slot small source spacer span strike strong style sub summary
   sup svg table tbody td template textarea tfoot th thead time title tr track
   tt u ul var video wbr xmp`.split(/\s+/),
);

const BLOCK_ELEMENTS = new Set(
  `address article aside blockquote br caption center dd details dialog dir div
   dl dt fieldset figcaption figure footer form h1 h2 h3 h4 h5 h6 header hgroup
   hr legend li main marquee menu nav ol p pre section summary table tbody td
   tfoot th thead tr ul`.split(/\s+/),
);
