import { expect, test } from "bun:test";
import { dependencyLinks } from "./extension-dependencies";

const library = {
  id: "0b8e7c52-3f7a-4f4e-9d0e-0d6f1c2a9b11",
  name: "LALib",
  creator: "someone",
};

test("pairs each named dependency with the extensions that match it", () => {
  const links = dependencyLinks(
    [{ text: "third-party/SillyTavern-LALib" }, { text: "vectors" }],
    [
      { name: "third-party/SillyTavern-LALib", works: [library] },
      { name: "vectors", works: [] },
    ],
  );

  expect(links).toEqual([
    { name: "third-party/SillyTavern-LALib", works: [library] },
    { name: "vectors", works: [] },
  ]);
});

test("shows the bare name when the page was read without matches", () => {
  expect(dependencyLinks([{ text: "third-party/Other" }], [])).toEqual([
    { name: "third-party/Other", works: [] },
  ]);
});
