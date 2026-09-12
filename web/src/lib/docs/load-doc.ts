import { readFile } from "node:fs/promises";
import path from "node:path";
import type { DocCollection, DocPage } from "./collections";
import { type Doc, readDoc } from "./read-doc";

const CONTENT_ROOT = path.join(process.cwd(), "content", "developers");

/** Reads one page of a collection from the Markdown checked in beside the site. */
export async function loadDoc(
  collection: DocCollection,
  page: DocPage,
): Promise<Doc> {
  const file = path.join(CONTENT_ROOT, collection.directory, page.file);
  return readDoc(await readFile(file, "utf8"));
}
