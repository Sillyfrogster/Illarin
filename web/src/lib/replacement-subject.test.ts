import { expect, test } from "bun:test";
import {
  replacementSubjectLabel,
  summariseReplacement,
} from "./replacement-subject";

test("names a role in the words the page uses", () => {
  expect(replacementSubjectLabel("example_dialogue")).toBe("Example dialogue");
});

test("says pictures and preserved data rather than their stored names", () => {
  expect(replacementSubjectLabel("images")).toBe("Pictures");
  expect(replacementSubjectLabel("opaque_data")).toBe("Preserved data");
});

test("counts repeated changes to one part into a single line", () => {
  const summary = summariseReplacement([
    { kind: "addition", subject: "greetings" },
    { kind: "addition", subject: "greetings" },
    { kind: "removal", subject: "greetings" },
    { kind: "change", subject: "description" },
  ]);

  expect(summary).toEqual([
    {
      subject: "greetings",
      label: "Greetings",
      detail: "2 added, 1 removed by the file",
      replacesYourEdit: false,
    },
    {
      subject: "description",
      label: "Description",
      detail: "Changed by the file",
      replacesYourEdit: false,
    },
  ]);
});

test("marks a part where the file replaces an edit of the creator's own", () => {
  const summary = summariseReplacement([
    { kind: "change", subject: "images" },
    { kind: "conflict", subject: "images" },
  ]);

  expect(summary[0]?.detail).toBe("1 changed by the file");
  expect(summary[0]?.replacesYourEdit).toBe(true);
});
