import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { ReplacementWarnings } from "./ReplacementWarnings";

test("replacement warnings survive a null conflict list and name missing wording", () => {
  const markup = renderToStaticMarkup(
    <ReplacementWarnings
      preview={{
        conflicts: null as never,
        missingWording: ["New private prompt"],
        seals: 1,
      }}
    />,
  );
  expect(markup).toContain("New private prompt");
  expect(markup).toContain("sealed and empty");
  expect(markup).not.toContain("overwrites edits");
});
