import { expect, test } from "bun:test";
import { groupsFor, sectionAt, sectionsFor } from "./sections";

test("a section is grouped, gated by role and matched by the address", () => {
  for (const role of ["moderator", "admin"] as const) {
    const sections = sectionsFor(role);
    expect(sections.map((section) => section.href)).toContain("/staff");
    expect(groupsFor(role).map(([group]) => group)).toEqual([
      ...new Set(sections.map((section) => section.group)),
    ]);
  }
  expect(sectionAt("moderator", "/staff")?.label).toBe("Report");
  expect(sectionAt("moderator", "/staff/nothing-here")).toBeUndefined();
});
