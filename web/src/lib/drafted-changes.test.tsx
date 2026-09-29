import { afterEach, expect, mock, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";

mock.module("next/navigation", () => ({ useRouter: () => ({ refresh() {} }) }));

const { DraftedChangesProvider, useDraftedChanges } = await import(
  "./drafted-changes"
);
const { saveWorkDetails } = await import("./api/query");

const WORK = "00000000-0000-4000-8000-000000000025";
const originalFetch = globalThis.fetch;

afterEach(() => {
  globalThis.fetch = originalFetch;
  Reflect.deleteProperty(globalThis, "window");
});

function EditorVersion() {
  return <p>editing from version {useDraftedChanges().version}</p>;
}

test("a page read older than this tab's last save never reaches the editor, as when Back replays the cached page", async () => {
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: { dispatchEvent: () => true },
  });
  globalThis.fetch = mock(
    async () =>
      new Response(null, {
        status: 204,
        headers: { "X-Drafted-Changes-Version": "12" },
      }),
  ) as unknown as typeof fetch;
  await saveWorkDetails({ workId: WORK, version: 11 }, WORK, {
    blurb: "Saved before leaving the page.",
    isNsfw: false,
    name: "Fixture work",
  });

  const replayed = renderToStaticMarkup(
    <DraftedChangesProvider version={11} workId={WORK}>
      <EditorVersion />
    </DraftedChangesProvider>,
  );
  const reread = renderToStaticMarkup(
    <DraftedChangesProvider version={12} workId={WORK}>
      <EditorVersion />
    </DraftedChangesProvider>,
  );

  expect(replayed).toBe("");
  expect(reread).toContain("editing from version 12");
});
