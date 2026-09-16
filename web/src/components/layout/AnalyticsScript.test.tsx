import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { AnalyticsScript } from "./AnalyticsScript";

test("a production page loads the tracker from its own origin without query strings or fragments", () => {
  const markup = renderToStaticMarkup(<AnalyticsScript production />);
  expect(markup).toContain('src="/stats/script.js"');
  expect(markup).toMatch(/data-website-id="[0-9a-f]{8}-[0-9a-f-]{27}"/);
  expect(markup).toContain('data-exclude-search="true"');
  expect(markup).toContain('data-exclude-hash="true"');
});

test("a development page loads no tracker", () => {
  expect(renderToStaticMarkup(<AnalyticsScript production={false} />)).toBe("");
});
