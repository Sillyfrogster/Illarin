import { type ApiMethod, type ApiOptions, type ApiResult, api } from "./client";

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export type Refusal = { error?: string; field?: string; version?: number };

export type Answer<T> = { value?: T; error?: string; refusal?: Refusal };

/** Calls a /v1 route from the browser and turns a refusal into a message a person can read. */
export async function ask<T>(
  method: ApiMethod,
  path: string,
  options: ApiOptions = {},
): Promise<Answer<T>> {
  let answer: ApiResult<T>;
  try {
    answer = await api<T>(method, `/v1${path}`, {
      cache: "no-store",
      ...options,
    });
  } catch {
    return { error: UNREACHABLE };
  }
  if (!answer.response.ok) {
    const body = (
      typeof answer.error === "object" && answer.error !== null
        ? answer.error
        : {}
    ) as Refusal;
    return {
      error: body.error ?? "That did not work. Try again.",
      refusal: body,
    };
  }
  return { value: answer.data };
}
