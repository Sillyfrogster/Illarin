import type { ManagedConnectedApp } from "@/lib/api/shapes";
import { isStringArray } from "./answer";
import { readableDate } from "./dates";
export type { ManagedConnectedApp };

export function revoked(
  app: ManagedConnectedApp,
  at: string,
): ManagedConnectedApp {
  return {
    ...app,
    acceptedFormats: [],
    appVersion: undefined,
    capabilities: [],
    protocolVersion: null,
    revokedAt: at,
  };
}

export function installedHere(app: ManagedConnectedApp): string | null {
  if (!app.permissions.includes("library:sync")) return null;
  if (app.installed === 0) return "No installed works reported";
  const held =
    app.installed === 1
      ? "1 work installed"
      : `${app.installed} works installed`;
  return app.updatesAvailable > 0
    ? `${held} · ${app.updatesAvailable} with a newer version here`
    : `${held} · all up to date`;
}

export function seenAt(app: ManagedConnectedApp): string {
  return app.lastSeenAt
    ? `last seen ${readableDate(app.lastSeenAt)}`
    : "not seen yet";
}

export function isConnectedAppList(
  value: unknown,
): value is { items: ManagedConnectedApp[] } {
  return (
    typeof value === "object" &&
    value !== null &&
    "items" in value &&
    Array.isArray(value.items) &&
    value.items.every(isConnectedApp)
  );
}

function isConnectedApp(value: unknown): value is ManagedConnectedApp {
  if (typeof value !== "object" || value === null) return false;
  const app = value as Record<string, unknown>;
  return (
    typeof app.id === "string" &&
    typeof app.appName === "string" &&
    typeof app.name === "string" &&
    (app.appVersion === undefined ||
      app.appVersion === null ||
      typeof app.appVersion === "string") &&
    (app.protocolVersion === null || typeof app.protocolVersion === "number") &&
    isStringArray(app.capabilities) &&
    isStringArray(app.acceptedFormats) &&
    typeof app.prefix === "string" &&
    isStringArray(app.permissions) &&
    typeof app.installed === "number" &&
    typeof app.updatesAvailable === "number" &&
    typeof app.connectedAt === "string" &&
    (app.lastSeenAt === null || typeof app.lastSeenAt === "string") &&
    (app.revokedAt === null || typeof app.revokedAt === "string")
  );
}
