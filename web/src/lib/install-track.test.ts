import { expect, test } from "bun:test";
import type { QueuedDelivery, WorkInstance } from "@/lib/api/query";
import { installTrack } from "@/lib/install-track";

function instance(overrides: Partial<WorkInstance> = {}): WorkInstance {
  return {
    instanceId: "i1",
    applicationName: "Lumiverse",
    instanceName: "desk",
    lastSeenAt: null,
    canReceive: true,
    reportsLibrary: true,
    delivery: null,
    installedVersion: null,
    updateAvailable: false,
    ...overrides,
  };
}

function delivery(overrides: Partial<QueuedDelivery> = {}): QueuedDelivery {
  return {
    id: "d1",
    instanceId: "i1",
    workId: "a1",
    state: "queued",
    queuedAt: "2026-09-13T10:00:00Z",
    settledAt: null,
    expiresAt: "2026-09-20T10:00:00Z",
    updatesInstall: false,
    ...overrides,
  };
}

test("an instance with nothing sent and nothing installed has no track", () => {
  expect(installTrack(instance())).toBeNull();
});

test("a queued delivery starts the track and keeps it live", () => {
  const track = installTrack(instance({ delivery: delivery() }));
  expect(track?.steps.map((step) => step.standing)).toEqual([
    "now",
    "later",
    "later",
  ]);
  expect(track?.note).toBe("Waiting for desk to collect it.");
  expect(track?.live).toBe(true);
});

test("a released delivery has been picked up", () => {
  const track = installTrack(
    instance({ delivery: delivery({ state: "released" }) }),
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
    instance({
      delivery: delivery({
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
    instance({
      delivery: delivery({
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

test("a stopped delivery shows its reason instead of progress", () => {
  const track = installTrack(
    instance({
      delivery: delivery({
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
      instance({
        delivery: delivery({
          state: "failed",
          reason: "withdrawn",
          settledAt: "2026-09-13T10:01:00Z",
        }),
      }),
    )?.stopped,
  ).toBe("This work was withdrawn before it could be collected.");
});

test("an install the library reports without a delivery on record is a standing line, not a track", () => {
  const track = installTrack(instance({ installedVersion: 1 }));
  expect(track?.steps).toEqual([]);
  expect(track?.note).toBe("Installed on desk.");
  expect(
    installTrack(instance({ installedVersion: 1, updateAvailable: true }))
      ?.note,
  ).toBe("Installed on desk, and a newer version exists here.");
});
