import { expect, test } from "bun:test";
import type { WorkDetail } from "./api/query";
import { workMetadata } from "./work-metadata";

const ID = "0f6b7a4c-3d21-4a5e-9c8b-1f2e3d4c5b6a";

function work(over: Partial<WorkDetail> = {}): WorkDetail {
  return {
    id: ID,
    type: "character",
    name: "Christine Novak",
    blurb: "She closes the book on a ribbon.",
    tags: [],
    creator: "nimhloth",
    extensionDependencies: [],
    installedAppVersions: [],
    readerApp: null,
    isNsfw: false,
    visibility: "listed",
    lifecycle: "published",
    isOwner: false,
    hasPrivatePrompts: false,
    allowedApps: [],
    eligibleApps: [],
    downloads: [],
    appFormats: [],
    original: null,
    createdAt: "2026-08-13T00:00:00Z",
    blocks: [],
    media: [],
    preview: "/media/aaaa/og/1",
    nsfwPreference: "blurred",
    ...over,
  };
}

test("calls a work with no name Untitled", () => {
  const meta = workMetadata(work({ name: "", blurb: "" }));

  expect(meta.title).toBe("Untitled · Character");
});

test("names the work and its type, and pitches it with the blurb", () => {
  const meta = workMetadata(work());

  expect(meta.title).toBe("Christine Novak · Character");
  expect(meta.description).toBe("She closes the book on a ribbon.");
  expect(meta.openGraph?.title).toBe("Christine Novak · Character");
  expect(meta.twitter && "card" in meta.twitter && meta.twitter.card).toBe(
    "summary_large_image",
  );
});

test("points the preview at the canonical address", () => {
  const meta = workMetadata(work());
  const canonical = `/a/${ID}/christine-novak`;

  expect(meta.alternates?.canonical).toBe(canonical);
  expect(meta.openGraph && "url" in meta.openGraph && meta.openGraph.url).toBe(
    canonical,
  );
});

test("uses the work card with or without a cover", () => {
  expect(workMetadata(work()).openGraph?.images).toEqual([
    {
      url: `/a/${ID}/card.png`,
      alt: "Christine Novak",
      width: 1200,
      height: 630,
    },
  ]);
  expect(workMetadata(work({ preview: null })).openGraph?.images).toEqual(
    workMetadata(work()).openGraph?.images,
  );
  expect(workMetadata(work()).twitter?.images).toEqual([`/a/${ID}/card.png`]);
});

test("draft metadata exposes neither its name nor a card", () => {
  const metadata = workMetadata(work({ lifecycle: "draft", isOwner: true }));
  expect(metadata.robots).toEqual({ index: false, follow: false });
  expect(JSON.stringify(metadata)).not.toContain("Christine");
  expect(metadata.openGraph).toBeUndefined();
});

test("stands in a description when the creator wrote no blurb", () => {
  expect(workMetadata(work({ blurb: "" })).description).toBe(
    "A character by nimhloth.",
  );
});

test("asks not to be indexed while unlisted, and still invites following", () => {
  expect(workMetadata(work({ visibility: "unlisted" })).robots).toEqual({
    index: false,
  });
  expect(workMetadata(work()).robots).toBeUndefined();
});

test("does not copy private prompt text into page or social metadata", () => {
  const privateText = "metadata-disclosure-canary-1c7ed05b";
  const metadata = workMetadata(
    work({
      type: "preset",
      hasPrivatePrompts: true,
      allowedApps: [{ id: "lumiverse", label: "Lumiverse" }],
      blocks: [
        {
          id: ID,
          definition: "preset_core",
          title: "Prompt",
          titleIsDefault: true,
          position: 0,
          hidden: false,
          layout: "single",
          width: "full",
          allowedLayouts: ["single"],
          required: true,
          hideable: false,
          isEmpty: false,
          elements: [
            {
              id: ID,
              type: "prompt_list",
              role: "prompt_fragments",
              slot: "main",
              label: "Prompt fragments",
              pinned: true,
              fromFile: false,
              isEmpty: false,
              facts: ["1 fragment"],
              content: {
                groups: [],
                fragments: [
                  {
                    id: ID,
                    name: "Private instructions",
                    role: "system",
                    text: privateText,
                    private: true,
                    enabled: true,
                  },
                ],
              },
            },
          ],
        },
      ],
    }),
  );

  expect(JSON.stringify(metadata)).not.toContain(privateText);
  expect(metadata.openGraph?.description).toBe(
    "She closes the book on a ribbon.",
  );
});
