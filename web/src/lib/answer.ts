/** What Illarin sends back when it refuses, which a page reads before it trusts anything else. */
export type Refusal = { error?: string; field?: string };

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

export function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) && value.every((item) => typeof item === "string")
  );
}
