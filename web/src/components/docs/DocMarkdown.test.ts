import { describe, expect, test } from "bun:test";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { DOCS, headingId, readDoc } from "@/lib/docs";
import { DocMarkdown, docSections } from "./DocMarkdown";

describe("published documentation links", () => {
  test("each section has a unique stable anchor", async () => {
    for (const doc of DOCS) {
      const source = await readDoc(doc.slug);
      expect(
        renderToStaticMarkup(createElement(DocMarkdown, { source })),
      ).toContain("<h1");
      const sections = docSections(source);
      const ids = [...source.matchAll(/^#{1,3} (.+)$/gm)].map(([, title]) =>
        headingId(title),
      );
      expect(ids.length).toBeGreaterThan(0);
      expect(sections.length).toBeGreaterThan(0);
      expect(new Set(ids).size).toBe(ids.length);
      for (const [, target] of source.matchAll(/\]\(#([^)]+)\)/g)) {
        expect(ids).toContain(target);
      }
    }
    expect(headingId("Extension install checklist")).toBe(
      "extension-install-checklist",
    );
  });
});
