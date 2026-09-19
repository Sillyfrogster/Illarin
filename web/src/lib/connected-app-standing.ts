import type { ManagedInstance } from "@/lib/api/shapes";
import { isStringArray } from "./answer";
import { readableDate } from "./dates";
export type { ManagedInstance };

export function revoked(
  instance: ManagedInstance,
  at: string,
): ManagedInstance {
  return {
    ...instance,
    acceptedTargets: [],
    applicationVersion: undefined,
    capabilities: [],
    protocolVersion: null,
    revokedAt: at,
  };
}

export function installedHere(instance: ManagedInstance): string | null {
  if (!instance.scopes.includes("library:sync")) return null;
  if (instance.installed === 0) return "No installed works reported";
  const held =
    instance.installed === 1
      ? "1 work installed"
      : `${instance.installed} works installed`;
  return instance.updatesAvailable > 0
    ? `${held} · ${instance.updatesAvailable} with a newer version here`
    : `${held} · all up to date`;
}

export function seenAt(instance: ManagedInstance): string {
  return instance.lastSeenAt
    ? `last seen ${readableDate(instance.lastSeenAt)}`
    : "not seen yet";
}

export function isInstanceList(
  value: unknown,
): value is { items: ManagedInstance[] } {
  return (
    typeof value === "object" &&
    value !== null &&
    "items" in value &&
    Array.isArray(value.items) &&
    value.items.every(isInstance)
  );
}

function isInstance(value: unknown): value is ManagedInstance {
  if (typeof value !== "object" || value === null) return false;
  const instance = value as Record<string, unknown>;
  return (
    typeof instance.id === "string" &&
    typeof instance.applicationName === "string" &&
    typeof instance.instanceName === "string" &&
    (instance.applicationVersion === undefined ||
      instance.applicationVersion === null ||
      typeof instance.applicationVersion === "string") &&
    (instance.protocolVersion === null ||
      typeof instance.protocolVersion === "number") &&
    isStringArray(instance.capabilities) &&
    isStringArray(instance.acceptedTargets) &&
    typeof instance.prefix === "string" &&
    isStringArray(instance.scopes) &&
    typeof instance.installed === "number" &&
    typeof instance.updatesAvailable === "number" &&
    typeof instance.linkedAt === "string" &&
    (instance.lastSeenAt === null || typeof instance.lastSeenAt === "string") &&
    (instance.revokedAt === null || typeof instance.revokedAt === "string")
  );
}
