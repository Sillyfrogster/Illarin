import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { RecordedVersion } from "@/lib/api/query";
import { UpdateHistory } from "./UpdateHistory";

const ASSET = "9b1f6c2e-4d3a-4f18-8c7b-2e5a1d9f3b40";

function version(over: Partial<RecordedVersion>): RecordedVersion {
  return {
    id: `version-${over.number ?? 1}`,
    number: 1,
    recordedAt: "2026-03-12T09:00:00Z",
    initial: false,
    versionLabel: "",
    summary: "",
    notes: "",
    ...over,
  };
}

const HISTORY = [
  version({ number: 3, summary: "Rewrote the opening", versionLabel: "1.2" }),
  version({ number: 2, summary: "Opened the east window" }),
  version({ number: 1, initial: true }),
];

function history(versions: RecordedVersion[], download?: string) {
  return renderToStaticMarkup(
    <UpdateHistory
      assetId={ASSET}
      download={download ? <a href={download}>Get character</a> : null}
      kind="character"
      versions={versions}
    />,
  );
}

test("the rail reaches every recorded version by its own address", () => {
  const html = history(HISTORY);

  for (const number of [3, 2, 1]) {
    expect(html).toContain(`href="#version-${number}"`);
    expect(html).toContain(`id="version-${number}"`);
  }
  expect(html).toContain('aria-label="Recorded versions"');
  expect(html).toContain('aria-controls="version-spine"');
});

test("only the newest version is the one readers have, and only it carries the download", () => {
  const html = history(HISTORY, `/download/${ASSET}/chara_card_v3`);

  expect(html.match(/Readers have this/g)).toHaveLength(1);
  expect(html.match(/Get character/g)).toHaveLength(1);
  const newest = html.indexOf('id="version-3"');
  const next = html.indexOf('id="version-2"');
  expect(html.indexOf("Get character")).toBeGreaterThan(newest);
  expect(html.indexOf("Get character")).toBeLessThan(next);
});

test("a version with nothing recorded before it offers no comparison", () => {
  const html = history([version({ number: 1, initial: true })]);

  expect(html).toContain("there is nothing to compare it with");
  expect(html).not.toContain("What changed since");
});

test("a later version compares itself with the version before it", () => {
  const html = history(HISTORY);

  expect(html).toContain("What changed since update 2");
  expect(html).toContain("What changed since initial recording");
});

test("an asset with no recorded versions says history begins at publication", () => {
  const html = history([]);

  expect(html).toContain("History begins at its first publication");
  expect(html).not.toContain("Recorded versions");
});
