import { expect, test } from "bun:test";
import {
  isWorkId,
  workHistoryHref,
  workHref,
  workRedirect,
  workSlug,
} from "./work-url";

const ID = "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a";
const CHRISSY = { id: ID, name: "Christine Novak" };

test("lowercases and joins words with single hyphens", () => {
  expect(workSlug("The Quiet Archivist")).toBe("the-quiet-archivist");
});

test("drops combining marks left by decomposition", () => {
  expect(workSlug("Chrissy Nóvak")).toBe("chrissy-novak");
  expect(workSlug("Ångström")).toBe("angstrom");
});

test("collapses runs of punctuation and trims the ends", () => {
  expect(workSlug("  ***Rain,  and No Umbrella!!!  ")).toBe(
    "rain-and-no-umbrella",
  );
});

test("has no slug for a name that normalizes to nothing", () => {
  expect(workSlug("日本語")).toBe("");
  expect(workSlug("!!!")).toBe("");
  expect(workSlug("")).toBe("");
});

test("has no slug for a name that would claim a segment Illarin owns", () => {
  expect(workSlug("History")).toBe("");
  expect(workHref(ID, "History")).toBe(`/a/${ID}`);
});

test("caps at sixty characters, preferring the last word boundary", () => {
  const slug = workSlug(
    "A brilliant unassuming girl who spent years being overlooked entirely",
  );

  expect(slug).toBe("a-brilliant-unassuming-girl-who-spent-years-being");
  expect(slug.length).toBeLessThanOrEqual(60);
  expect(slug.endsWith("-")).toBe(false);
});

test("cuts mid-word only when the first sixty characters hold no boundary", () => {
  expect(workSlug("a".repeat(80))).toBe("a".repeat(60));
});

test("transliterates nothing", () => {
  expect(workSlug("Ярославль")).toBe("");
  expect(workSlug("Straße")).toBe("stra-e");
});

test("addresses a work by id, with the slug only when there is one", () => {
  expect(workHref(ID, "The Quiet Archivist")).toBe(
    `/a/${ID}/the-quiet-archivist`,
  );
  expect(workHref(ID, "日本語")).toBe(`/a/${ID}`);
});

test("sends a bare address to the slugged one", () => {
  expect(workRedirect({ id: ID }, CHRISSY)).toBe(`/a/${ID}/christine-novak`);
});

test("sends a stale or wrong slug to the current one", () => {
  expect(workRedirect({ id: ID, slug: ["chrissy-old-name"] }, CHRISSY)).toBe(
    `/a/${ID}/christine-novak`,
  );
  expect(workRedirect({ id: ID, slug: ["christine", "novak"] }, CHRISSY)).toBe(
    `/a/${ID}/christine-novak`,
  );
});

test("leaves the canonical address alone", () => {
  expect(workRedirect({ id: ID, slug: ["christine-novak"] }, CHRISSY)).toBe(
    null,
  );
});

test("makes the bare address canonical when there is no slug", () => {
  const unnamed = { id: ID, name: "日本語" };

  expect(workRedirect({ id: ID }, unnamed)).toBe(null);
  expect(workRedirect({ id: ID, slug: ["something"] }, unnamed)).toBe(
    `/a/${ID}`,
  );
});

test("recognizes a work id, and nothing else, as one", () => {
  expect(isWorkId(ID)).toBe(true);
  expect(isWorkId(ID.toUpperCase())).toBe(true);
  expect(isWorkId("christine-novak")).toBe(false);
  expect(isWorkId(`${ID}x`)).toBe(false);
  expect(isWorkId("")).toBe(false);
});

test("opens update history on the work page, at one version when asked", () => {
  expect(workHistoryHref(ID, "The Glass Cartographer")).toBe(
    `/a/${ID}/the-glass-cartographer?history`,
  );
  expect(workHistoryHref(ID, "", 3)).toBe(`/a/${ID}?history#version-3`);
});
