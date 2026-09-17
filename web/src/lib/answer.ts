export type Refusal = { error?: string; field?: string };

/** Reads a refusal body, throwing when it was not JSON so the caller treats the API as unreachable. */
export function readRefusal(error: unknown): Refusal {
  if (typeof error === "object" && error !== null) return error as Refusal;
  throw new Error("The refusal could not be read.");
}

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
