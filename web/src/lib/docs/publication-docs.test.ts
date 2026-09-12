import { expect, test } from "bun:test";
import { PUBLICATION_DOCS } from "./collections";
import { type Contract, checkDoc } from "./contract";
import { loadDoc } from "./load-doc";

const CONTRACT_FILE = "../api/openapi/openapi.gen.yaml";

async function contract(): Promise<Contract> {
  return Bun.YAML.parse(await Bun.file(CONTRACT_FILE).text()) as Contract;
}

test("every page of the publication reference reads and carries its title", async () => {
  for (const page of PUBLICATION_DOCS.pages) {
    const doc = await loadDoc(PUBLICATION_DOCS, page);
    expect(doc.title).toBe(
      page.title === "Overview" ? "Publication API" : page.title,
    );
    expect(doc.lede.length).toBeGreaterThan(0);
  }
});

test("every example in the publication reference agrees with the contract", async () => {
  const agreed = await contract();
  for (const page of PUBLICATION_DOCS.pages) {
    const doc = await loadDoc(PUBLICATION_DOCS, page);
    expect(checkDoc(doc, agreed)).toEqual([]);
  }
});

test("the reference links only to pages that exist", async () => {
  const known = new Set(
    PUBLICATION_DOCS.pages.map((page) =>
      page.slug
        ? `${PUBLICATION_DOCS.href}/${page.slug}`
        : PUBLICATION_DOCS.href,
    ),
  );
  for (const page of PUBLICATION_DOCS.pages) {
    const doc = await loadDoc(PUBLICATION_DOCS, page);
    const source = JSON.stringify(doc.blocks);
    for (const [, href] of source.matchAll(/"href":"(\/developers[^"#]*)/g)) {
      expect(known.has(href)).toBe(true);
    }
  }
});
