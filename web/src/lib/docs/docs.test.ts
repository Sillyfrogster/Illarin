import { expect, test } from "bun:test";
import { DOCS, docHref } from "./collections";
import { type Contract, checkDoc } from "./contract";
import { loadDoc } from "./load-doc";

const CONTRACT_FILE = "../api/openapi/openapi.gen.yaml";

async function contract(): Promise<Contract> {
  return Bun.YAML.parse(await Bun.file(CONTRACT_FILE).text()) as Contract;
}

const PAGES = DOCS.flatMap((collection) =>
  collection.pages.map((page) => ({ collection, page })),
);

test("every page reads and carries its title", async () => {
  for (const { collection, page } of PAGES) {
    const doc = await loadDoc(collection, page);
    expect(doc.title).toBe(page.slug === "" ? collection.name : page.title);
    expect(doc.lede.length).toBeGreaterThan(0);
  }
});

test("every example agrees with the contract", async () => {
  const agreed = await contract();
  for (const { collection, page } of PAGES) {
    const doc = await loadDoc(collection, page);
    expect({
      page: docHref(collection, page),
      problems: checkDoc(doc, agreed),
    }).toEqual({ page: docHref(collection, page), problems: [] });
  }
});

test("the pages link only to pages that exist", async () => {
  const known = new Set(
    PAGES.map(({ collection, page }) => docHref(collection, page)),
  );
  for (const { collection, page } of PAGES) {
    const doc = await loadDoc(collection, page);
    const source = JSON.stringify(doc.blocks);
    for (const [, href] of source.matchAll(/"href":"(\/developers[^"#]*)/g)) {
      expect(known).toContain(href);
    }
  }
});

test("every collection lives at its own address under the developer site", () => {
  const addresses = DOCS.map((collection) => collection.href);
  expect(new Set(addresses).size).toBe(addresses.length);
  for (const collection of DOCS) {
    expect(collection.href).toBe(`/developers/${collection.directory}`);
  }
});
