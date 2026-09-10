import type { components } from "@/lib/api/schema";

export type PendingLink = components["schemas"]["PendingLink"];
export type PendingDeviceLink = components["schemas"]["PendingDeviceLink"];
export type LinkRedirect = components["schemas"]["LinkRedirect"];

/** Reads the body of an answer that may carry nothing at all. */
export async function readJSON(response: Response): Promise<unknown> {
  if (response.status === 204) return null;
  try {
    return await response.json();
  } catch {
    return null;
  }
}

/** Prefers Illarin's own reason for a refusal over the one the page would guess. */
export function refusalMessage(value: unknown, fallback: string) {
  if (
    typeof value === "object" &&
    value !== null &&
    "error" in value &&
    typeof value.error === "string" &&
    value.error.trim()
  ) {
    return value.error;
  }
  return fallback;
}

/** A link request is only shown once every field the decision rests on is present. */
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

/**
 * An approved browser link hands the reader back to the application through a
 * callback the application chose, so the address is opened only when it reaches
 * a port on the reader's own machine and carries nothing else.
 */
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

/** The moment a request stops being available, written the way a reader reads a clock. */
export function readableExpiry(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) && value.every((item) => typeof item === "string")
  );
}
