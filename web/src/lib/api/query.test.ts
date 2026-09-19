import { afterEach, describe, expect, mock, test } from "bun:test";
import { DRAFTED_CHANGES_STALE } from "@/lib/drafted-changes";
import { saveWorkDetails } from "./query";

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

describe("details client", () => {
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

      await saveWorkDetails({ version: 7 }, ID, {
        blurb,
        isNsfw: false,
        name: "Fixture work",
      });

      expect(sent).toBeDefined();
      expect(await sent?.clone().json()).toEqual({
        blurb,
        isNsfw: false,
        name: "Fixture work",
      });
      expect(sent?.headers.get("X-Drafted-Changes-Version")).toBe("7");
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
            code: "drafted_changes_conflict",
            currentVersion: 8,
            error: "This working copy changed after you opened it.",
          },
          { status: 409 },
        ),
      ),
    );
    const details = {
      blurb: "Keep this unsaved pitch.",
      isNsfw: false,
      name: "Fixture work",
    };

    await expect(saveWorkDetails({ version: 7 }, ID, details)).rejects.toThrow(
      "This working copy changed after you opened it.",
    );

    expect(events).toEqual([DRAFTED_CHANGES_STALE]);
    expect(details.blurb).toBe("Keep this unsaved pitch.");
  });
});
