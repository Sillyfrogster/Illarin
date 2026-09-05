import { expect, test } from "bun:test";
import { archivePage } from "./publication-metadata";

test("an archive address with no page segment is page one", () => {
  expect(archivePage(undefined)).toBe(1);
  expect(archivePage([])).toBe(1);
});

test("a numbered archive address reads its page", () => {
  expect(archivePage(["page", "2"])).toBe(2);
  expect(archivePage(["page", "40"])).toBe(40);
});

test("anything but a whole page number counting from one is refused", () => {
  expect(archivePage(["page"])).toBeNull();
  expect(archivePage(["page", "0"])).toBeNull();
  expect(archivePage(["page", "-1"])).toBeNull();
  expect(archivePage(["page", "01"])).toBeNull();
  expect(archivePage(["page", "2.5"])).toBeNull();
  expect(archivePage(["page", "two"])).toBeNull();
  expect(archivePage(["archive", "2"])).toBeNull();
  expect(archivePage(["page", "2", "3"])).toBeNull();
});
