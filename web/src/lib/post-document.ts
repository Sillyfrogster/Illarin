/** The document version this build writes. Go refuses anything else. */
export const POST_DOCUMENT_VERSION = 1;

export type PostMark =
  | { type: "bold" }
  | { type: "italic" }
  | { type: "code" }
  | { type: "link"; href: string };

export type PostSpan = { type: "text"; text: string; marks?: PostMark[] };

export type PostItem = { type: "listItem"; content: PostBlock[] };

export type PostBlock =
  | { type: "paragraph"; content: PostSpan[] }
  | { type: "heading"; level: number; content: PostSpan[] }
  | { type: "bulletList"; content: PostItem[] }
  | { type: "orderedList"; content: PostItem[] }
  | { type: "quote"; content: PostBlock[] }
  | { type: "divider" };

export type PostDocument = { version: number; content: PostBlock[] };

export function emptyPostDocument(): PostDocument {
  return { version: POST_DOCUMENT_VERSION, content: [] };
}

/** Whether a document holds anything a reader would see. */
export function isWritten(document: PostDocument): boolean {
  return textLength(document.content) > 0;
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
        for (const item of block.content) total += textLength(item.content);
        break;
      case "quote":
        total += textLength(block.content);
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
