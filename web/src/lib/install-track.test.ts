import { expect, test } from "bun:test";
import type { QueuedSend, WorkConnectedApp } from "@/lib/api/query";
import { installTrack } from "@/lib/install-track";

function connectedApp(
  overrides: Partial<WorkConnectedApp> = {},
): WorkConnectedApp {
  return {
    connectedAppId: "i1",
    appName: "Lumiverse",
    name: "desk",
    lastSeenAt: null,
    canReceive: true,
    reportsLibrary: true,
    send: null,
    installedVersion: null,
    updateAvailable: false,
    ...overrides,
  };
}

function send(overrides: Partial<QueuedSend> = {}): QueuedSend {
  return {
    id: "d1",
    connectedAppId: "i1",
    workId: "a1",
    state: "queued",
    queuedAt: "2026-09-13T10:00:00Z",
    settledAt: null,
    expiresAt: "2026-09-20T10:00:00Z",
    updatesInstall: false,
    ...overrides,
  };
}

test("a connected app with nothing sent and nothing installed has no track", () => {
  expect(installTrack(connectedApp())).toBeNull();
});

test("a queued send starts the track and keeps it live", () => {
  const track = installTrack(connectedApp({ send: send() }));
  expect(track?.steps.map((step) => step.standing)).toEqual([
    "now",
    "later",
    "later",
  ]);
  expect(track?.note).toBe("Waiting for desk to collect it.");
  expect(track?.live).toBe(true);
});

test("a released send has been picked up", () => {
  const track = installTrack(
    connectedApp({ send: send({ state: "released" }) }),
  );
  expect(track?.steps.map((step) => step.standing)).toEqual([
    "done",
    "now",
    "later",
  ]);
  expect(track?.note).toBe("desk picked it up and is installing it.");
  expect(track?.live).toBe(true);
});

test("an acknowledged first install waits for approval", () => {
  const track = installTrack(
    connectedApp({
      send: send({
        state: "delivered",
        settledAt: "2026-09-13T10:01:00Z",
      }),
      installedVersion: 1,
    }),
  );
  expect(track?.steps.map((step) => step.standing)).toEqual([
    "done",
    "done",
    "done",
  ]);
  expect(track?.steps[2].label).toBe("Installed");
  expect(track?.note).toBe(
    "Installed on desk, switched off until you approve its permissions in Lumiverse.",
  );
  expect(track?.live).toBe(false);
});

test("an update keeps the extension on and says so", () => {
  const track = installTrack(
    connectedApp({
      send: send({
        state: "delivered",
        settledAt: "2026-09-13T10:01:00Z",
        updatesInstall: true,
      }),
      installedVersion: 2,
    }),
  );
  expect(track?.steps[2].label).toBe("Updated");
  expect(track?.note).toBe(
    "Updated on desk. It stays on, and Lumiverse asks only about permissions the update adds.",
  );
});

test("a stopped send shows its reason instead of progress", () => {
  const track = installTrack(
    connectedApp({
      send: send({
        state: "failed",
        reason: "unsupported",
        settledAt: "2026-09-13T10:01:00Z",
      }),
    }),
  );
  expect(track?.steps.map((step) => step.standing)).toEqual([
    "later",
    "later",
    "later",
  ]);
  expect(track?.stopped).toBe("desk no longer says it installs extensions.");
  expect(track?.live).toBe(false);
  expect(
    installTrack(
      connectedApp({
        send: send({
          state: "failed",
          reason: "withdrawn",
          settledAt: "2026-09-13T10:01:00Z",
        }),
      }),
    )?.stopped,
  ).toBe("This work was withdrawn before it could be collected.");
});

test("an install the library reports without a send on record is a standing line, not a track", () => {
  const track = installTrack(connectedApp({ installedVersion: 1 }));
  expect(track?.steps).toEqual([]);
  expect(track?.note).toBe("Installed on desk.");
  expect(
    installTrack(connectedApp({ installedVersion: 1, updateAvailable: true }))
      ?.note,
  ).toBe("Installed on desk, and a newer version exists here.");
});
