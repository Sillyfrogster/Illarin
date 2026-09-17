import { afterEach, describe, expect, mock, test } from "bun:test";
import { WORKING_COPY_STALE } from "@/lib/working-copy";
import { saveAssetIdentity } from "./query";

const ID = "00000000-0000-4000-8000-000000000024";
const originalFetch = globalThis.fetch;

function useFetch(
  fetchImplementation: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>,
) {
  Object.defineProperty(globalThis, "fetch", {
    configurable: true,
    value: fetchImplementation,
    writable: true,
  });
}

afterEach(() => {
  useFetch(originalFetch);
  Reflect.deleteProperty(globalThis, "window");
});

describe("identity client", () => {
  for (const [action, blurb] of [
    ["add", "A new pitch."],
    ["edit", "A clearer pitch."],
    ["clear", ""],
  ]) {
    test(`sends the blurb when a creator ${action}s it`, async () => {
      let sent: Request | undefined;
      useFetch(
        mock(async (input: RequestInfo | URL, init?: RequestInit) => {
          sent = new Request(input, init);
          return new Response(null, { status: 204 });
        }),
      );

      await saveAssetIdentity({ version: 7 }, ID, {
        blurb,
        isNsfw: false,
        name: "Fixture asset",
      });

      expect(sent).toBeDefined();
      expect(await sent?.clone().json()).toEqual({
        blurb,
        isNsfw: false,
        name: "Fixture asset",
      });
      expect(sent?.headers.get("X-Working-Copy-Version")).toBe("7");
    });
  }

  test("reports a working-copy conflict without changing the request", async () => {
    const events: string[] = [];
    Object.defineProperty(globalThis, "window", {
      configurable: true,
      value: {
        dispatchEvent(event: Event) {
          events.push(event.type);
          return true;
        },
      },
    });
    useFetch(
      mock(async () =>
        Response.json(
          {
            code: "working_copy_conflict",
            currentVersion: 8,
            error: "This working copy changed after you opened it.",
          },
          { status: 409 },
        ),
      ),
    );
    const identity = {
      blurb: "Keep this unsaved pitch.",
      isNsfw: false,
      name: "Fixture asset",
    };

    await expect(
      saveAssetIdentity({ version: 7 }, ID, identity),
    ).rejects.toThrow("This working copy changed after you opened it.");

    expect(events).toEqual([WORKING_COPY_STALE]);
    expect(identity.blurb).toBe("Keep this unsaved pitch.");
  });
});
