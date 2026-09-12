import { browserFetch } from "./browser-mutation";

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export type Refusal = { error?: string; field?: string; version?: number };

export type Answer<T> = { value?: T; error?: string; refusal?: Refusal };

export async function ask<T>(
  path: string,
  init: RequestInit = {},
  read: (response: Response) => Promise<T>,
): Promise<Answer<T>> {
  let response: Response;
  try {
    response = await browserFetch(`/api/v1${path}`, {
      credentials: "same-origin",
      cache: "no-store",
      ...init,
    });
  } catch {
    return { error: UNREACHABLE };
  }
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as Refusal;
    return {
      error: body.error ?? "That did not work. Try again.",
      refusal: body,
    };
  }
  return { value: await read(response) };
}

export function json<T>(path: string, method: string, body?: unknown) {
  return ask<T>(
    path,
    {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    },
    (response) => response.json() as Promise<T>,
  );
}
