import { expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { renderToStaticMarkup } from "react-dom/server";
import {
  fromEditor,
  toEditor,
} from "@/components/publication/editor/tiptap-document";
import type { PostMedia } from "@/lib/api/query";
import { POST_DOCUMENT_VERSION, type PostDocument } from "@/lib/post-document";
import { PostBody } from "./PostBody";

/** The one corpus Go converts Markdown into and the site reads back. */
const IMPORTS = join(
  import.meta.dir,
  "../../../../api/internal/postdoc/testdata/markdown/carried",
);

const PLACED = "6f2c1b40-9d38-4a7e-b512-0c8e37a41d59";

const HELD: PostMedia[] = [
  {
    id: PLACED,
    postId: "b8d0f1a2-3c45-4e67-89ab-cdef01234567",
    purpose: "document",
    url: `/media/${PLACED}/detail/1`,
    thumbUrl: `/media/${PLACED}/grid/1`,
    width: 1200,
    height: 600,
  },
];

type ImportCase = { note: string; markdown: string; document: PostDocument };

const CARRIED: [string, ImportCase][] = readdirSync(IMPORTS).map((name) => [
  name,
  JSON.parse(readFileSync(join(IMPORTS, name), "utf8")) as ImportCase,
]);

test("the imports are there to read", () => {
  expect(CARRIED.length).toBeGreaterThan(0);
});

for (const [name, one] of CARRIED) {
  test(`${name} is written at the version this build emits`, () => {
    expect(one.document.version).toBe(POST_DOCUMENT_VERSION);
  });

  test(`${name} survives a round trip through the editor`, () => {
    expect(fromEditor(toEditor(one.document, HELD))).toEqual(one.document);
  });

  test(`${name} renders without stored markup`, () => {
    const html = renderToStaticMarkup(
      <PostBody document={one.document} media={HELD} />,
    );
    expect(html).not.toContain("&lt;");
    for (const mark of ["## ", "**", "~~", "](media:"]) {
      expect(html).not.toContain(mark);
    }
  });
}
