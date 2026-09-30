import { describe, expect, test } from "bun:test";
import { chooseTag, tagFragment, tagSearchHref } from "./tag-search";

describe("tag search", () => {
  test("links a tag to a tag: search, quoting a tag with a space", () => {
    expect(tagSearchHref("fantasy")).toBe("/browse?q=tag%3Afantasy");
    expect(tagSearchHref("slow burn")).toBe(
      `/browse?q=${encodeURIComponent('tag:"slow burn"')}`,
    );
  });

  test("reads the word being typed as a tag fragment", () => {
    expect(tagFragment("fan")).toBe("fan");
    expect(tagFragment("elf Fan")).toBe("fan");
    expect(tagFragment("tag:fan")).toBe("fan");
    expect(tagFragment('tag:"slow bu')).toBe("slow bu");
  });

  test("offers nothing for a finished word, a short one or an author", () => {
    expect(tagFragment("")).toBeNull();
    expect(tagFragment("elf ")).toBeNull();
    expect(tagFragment("f")).toBeNull();
    expect(tagFragment("author:ali")).toBeNull();
    expect(tagFragment("tag:")).toBeNull();
  });

  test("replaces only the word being typed with the chosen tag", () => {
    expect(chooseTag("fan", "fantasy")).toBe("tag:fantasy");
    expect(chooseTag("elf tag:fa", "fantasy")).toBe("elf tag:fantasy");
    expect(chooseTag('moon tag:"slow bu', "slow burn")).toBe(
      'moon tag:"slow burn"',
    );
  });
});
