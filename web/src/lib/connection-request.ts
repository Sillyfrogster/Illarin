import type {
  ConnectionRedirect,
  PendingCodeConnection,
  PendingConnection,
} from "@/lib/api/shapes";
import { isStringArray } from "./answer";
export type { ConnectionRedirect, PendingCodeConnection, PendingConnection };

export function isPendingConnection(
  value: unknown,
): value is PendingConnection {
  if (typeof value !== "object" || value === null) return false;
  const connection = value as Record<string, unknown>;
  return (
    typeof connection.appName === "string" &&
    typeof connection.name === "string" &&
    (connection.appVersion === undefined ||
      connection.appVersion === null ||
      typeof connection.appVersion === "string") &&
    typeof connection.protocolVersion === "number" &&
    isStringArray(connection.capabilities) &&
    isStringArray(connection.acceptedFormats) &&
    isStringArray(connection.permissions) &&
    typeof connection.expiresAt === "string"
  );
}

export function isPendingCodeConnection(
  value: unknown,
): value is PendingCodeConnection {
  return (
    isPendingConnection(value) &&
    "approvalToken" in value &&
    typeof value.approvalToken === "string"
  );
}

export function isConnectionRedirect(
  value: unknown,
): value is ConnectionRedirect {
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
