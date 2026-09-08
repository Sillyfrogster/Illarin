import { expect, test } from "bun:test";
import { archiveSteps } from "./archive-pages";

test("a single page offers no numbered navigation at all", () => {
  expect(archiveSteps(1, 1)).toEqual([]);
  expect(archiveSteps(1, 0)).toEqual([]);
});

test("a short archive lists every page it has", () => {
  expect(archiveSteps(1, 4)).toEqual([1, 2, 3, 4]);
  expect(archiveSteps(3, 5)).toEqual([1, 2, 3, 4, 5]);
});

test("a long archive keeps the first, the last and the pages beside this one", () => {
  expect(archiveSteps(6, 11)).toEqual([1, "gap", 5, 6, 7, "gap", 11]);
});

test("a gap stands only where more than one page was skipped", () => {
  expect(archiveSteps(1, 6)).toEqual([1, 2, "gap", 6]);
  expect(archiveSteps(2, 6)).toEqual([1, 2, 3, "gap", 6]);
  expect(archiveSteps(6, 6)).toEqual([1, "gap", 5, 6]);
});

test("one skipped page is shown rather than hidden behind a gap", () => {
  expect(archiveSteps(3, 6)).toEqual([1, 2, 3, 4, 5, 6]);
});

test("every step is a page a reader could actually ask for", () => {
  for (let page = 1; page <= 9; page++) {
    const steps = archiveSteps(page, 9);
    const numbers = steps.filter((step): step is number => step !== "gap");
    expect(numbers).toContain(page);
    expect(numbers).toContain(1);
    expect(numbers).toContain(9);
    expect(numbers).toEqual([...numbers].sort((a, b) => a - b));
    expect(new Set(numbers).size).toBe(numbers.length);
  }
});
