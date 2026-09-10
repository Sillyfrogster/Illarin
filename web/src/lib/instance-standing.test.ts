import { expect, test } from "bun:test";
import type { ManagedInstance } from "./instance-standing";
import {
  installedHere,
  isInstanceList,
  revoked,
  seenAt,
} from "./instance-standing";

const instance = (rest: Partial<ManagedInstance> = {}): ManagedInstance => ({
  acceptedTargets: ["chub"],
  applicationName: "Rookery",
  applicationVersion: "2.1.0",
  capabilities: ["import"],
  id: "00000000-0000-4000-8000-000000000001",
  installed: 0,
  instanceName: "Rookery on the study desk",
  lastSeenAt: null,
  linkedAt: "2026-09-01T10:00:00Z",
  prefix: "ill_abc",
  protocolVersion: 1,
  revokedAt: null,
  scopes: ["asset:receive"],
  updatesAvailable: 0,
  ...rest,
});

test("a revoked instance keeps its name and loses everything it reported", () => {
  const cut = revoked(instance(), "2026-09-10T09:00:00Z");

  expect(cut.revokedAt).toBe("2026-09-10T09:00:00Z");
  expect(cut.applicationName).toBe("Rookery");
  expect(cut.applicationVersion).toBeNull();
  expect(cut.protocolVersion).toBeNull();
  expect(cut.capabilities).toEqual([]);
  expect(cut.acceptedTargets).toEqual([]);
});

test("revoking one instance does not touch the one it was read from", () => {
  const one = instance();
  revoked(one, "2026-09-10T09:00:00Z");

  expect(one.revokedAt).toBeNull();
  expect(one.capabilities).toEqual(["import"]);
});

test("an instance that does not report a library says nothing about one", () => {
  expect(installedHere(instance())).toBeNull();
});

test("an instance that reports a library but holds nothing says so", () => {
  expect(installedHere(instance({ scopes: ["library:sync"] }))).toBe(
    "Nothing reported installed here yet",
  );
});

test("counts what an instance holds and whether Illarin has newer", () => {
  expect(
    installedHere(instance({ installed: 1, scopes: ["library:sync"] })),
  ).toBe("1 asset installed · all up to date");
  expect(
    installedHere(
      instance({
        installed: 12,
        scopes: ["library:sync"],
        updatesAvailable: 3,
      }),
    ),
  ).toBe("12 assets installed · 3 with a newer version here");
});

test("an instance never seen says that rather than leaving a gap", () => {
  expect(seenAt(instance())).toBe("not seen yet");
});

test("a revoked list is read back as a list", () => {
  expect(isInstanceList({ items: [instance()] })).toBe(true);
  expect(isInstanceList({ items: [{ id: "one" }] })).toBe(false);
  expect(isInstanceList({})).toBe(false);
  expect(isInstanceList(null)).toBe(false);
});
