/** The document version this build writes. Go refuses anything else. */
export const POST_DOCUMENT_VERSION = 2;

/** Every language a code block may be labelled with. Go holds the same list. */
export const POST_LANGUAGES = [
  "plain",
  "bash",
  "css",
  "diff",
  "go",
  "html",
  "javascript",
  "json",
  "markdown",
  "python",
  "rust",
  "sql",
  "toml",
  "typescript",
  "yaml",
] as const;

/** Every kind of aside a callout may be. Go holds the same list. */
export const POST_CALLOUT_KINDS = [
  "note",
  "tip",
  "important",
  "warning",
] as const;

export type PostLanguage = (typeof POST_LANGUAGES)[number];

export type PostCalloutKind = (typeof POST_CALLOUT_KINDS)[number];

export type PostMark =
  | { type: "bold" }
  | { type: "italic" }
  | { type: "strike" }
  | { type: "code" }
  | { type: "link"; href: string };

export type PostSpan = { type: "text"; text: string; marks?: PostMark[] };

export type PostItem = { type: "listItem"; content: PostBlock[] };

export type PostTask = {
  type: "taskItem";
  done: boolean;
  content: PostBlock[];
};

export type PostCell = {
  type: "tableCell";
  heading?: true;
  content: PostBlock[];
};

export type PostRow = { type: "tableRow"; content: PostCell[] };

export type PostBlock =
  | { type: "paragraph"; content: PostSpan[] }
  | { type: "heading"; level: number; anchor?: string; content: PostSpan[] }
  | { type: "bulletList"; content: PostItem[] }
  | { type: "orderedList"; content: PostItem[] }
  | { type: "taskList"; content: PostTask[] }
  | { type: "quote"; content: PostBlock[] }
  | { type: "codeBlock"; language: PostLanguage; source: string }
  | { type: "table"; content: PostRow[] }
  | { type: "callout"; kind: PostCalloutKind; content: PostBlock[] }
  | { type: "divider" };

export type PostDocument = { version: number; content: PostBlock[] };

export function emptyPostDocument(): PostDocument {
  return { version: POST_DOCUMENT_VERSION, content: [] };
}

/** Whether a document holds anything a reader would see. */
export function isWritten(document: PostDocument): boolean {
  return textLength(document.content) > 0;
}

export function isPostLanguage(name: string): name is PostLanguage {
  return (POST_LANGUAGES as readonly string[]).includes(name);
}

export function isPostCalloutKind(kind: string): kind is PostCalloutKind {
  return (POST_CALLOUT_KINDS as readonly string[]).includes(kind);
}

function textLength(blocks: PostBlock[]): number {
  let total = 0;
  for (const block of blocks) {
    switch (block.type) {
      case "paragraph":
      case "heading":
        for (const span of block.content) total += span.text.trim().length;
        break;
      case "bulletList":
      case "orderedList":
      case "taskList":
        for (const entry of block.content) total += textLength(entry.content);
        break;
      case "quote":
      case "callout":
        total += textLength(block.content);
        break;
      case "codeBlock":
        total += block.source.trim().length;
        break;
      case "table":
        for (const row of block.content) {
          for (const cell of row.content) total += textLength(cell.content);
        }
        break;
    }
  }
  return total;
}

/** Go validated the stored body, so the reader takes it at its word. */
export function asPostDocument(value: {
  version: number;
  content: Record<string, unknown>[];
}): PostDocument {
  return value as unknown as PostDocument;
}
