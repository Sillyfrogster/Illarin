import type { components } from "@/lib/api/schema";
import { isStringArray } from "./answer";

export type PendingLink = components["schemas"]["PendingLink"];
export type PendingDeviceLink = components["schemas"]["PendingDeviceLink"];
export type LinkRedirect = components["schemas"]["LinkRedirect"];

export function isPendingLink(value: unknown): value is PendingLink {
  if (typeof value !== "object" || value === null) return false;
  const link = value as Record<string, unknown>;
  return (
    typeof link.applicationName === "string" &&
    typeof link.instanceName === "string" &&
    (link.applicationVersion === undefined ||
      link.applicationVersion === null ||
      typeof link.applicationVersion === "string") &&
    typeof link.protocolVersion === "number" &&
    isStringArray(link.capabilities) &&
    isStringArray(link.acceptedTargets) &&
    isStringArray(link.scopes) &&
    typeof link.expiresAt === "string"
  );
}

export function isPendingDeviceLink(
  value: unknown,
): value is PendingDeviceLink {
  return (
    isPendingLink(value) &&
    "approvalToken" in value &&
    typeof value.approvalToken === "string"
  );
}

export function isLinkRedirect(value: unknown): value is LinkRedirect {
  return (
    typeof value === "object" &&
    value !== null &&
    "redirectUrl" in value &&
    typeof value.redirectUrl === "string"
  );
}

export function isSafeLoopbackRedirect(value: string) {
  const authority =
    /^http:\/\/(?:127\.0\.0\.1|\[::1\]):([0-9]{1,5})(?:[/?#]|$)/i.exec(value);
  if (!authority) return false;
  const port = Number(authority[1]);
  if (!Number.isInteger(port) || port < 1 || port > 65_535) return false;

  try {
    const url = new URL(value);
    return (
      url.protocol === "http:" &&
      (url.hostname === "127.0.0.1" || url.hostname === "[::1]") &&
      !url.username &&
      !url.password &&
      !url.hash
    );
  } catch {
    return false;
  }
}

export function readableExpiry(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
