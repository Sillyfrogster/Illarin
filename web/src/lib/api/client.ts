import { markBrowserMutation } from "./browser-mutation";

const baseUrl =
  typeof window === "undefined"
    ? (process.env.API_URL ?? "http://localhost:8080")
    : "/api";

export type ApiMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export type ApiOptions = {
  body?: unknown;
  query?: object;
  headers?: HeadersInit;
  signal?: AbortSignal;
  cache?: RequestCache;
  keepalive?: boolean;
};

/** The typed body when the call succeeded, or the parsed error body when it did not. */
export type ApiResult<T> =
  | { data: T; error?: undefined; response: Response }
  | { data?: undefined; error: unknown; response: Response };

/** Calls one API route and reads its answer, throwing when the API cannot be reached. */
export async function api<T>(
  method: ApiMethod,
  path: string,
  options: ApiOptions = {},
): Promise<ApiResult<T>> {
  const { body, query, signal, cache, keepalive } = options;
  const headers = new Headers(options.headers);
  const isForm = body instanceof FormData;
  if (body !== undefined && !isForm && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  markBrowserMutation(method, headers);

  const response = await fetch(`${baseUrl}${path}${searchOf(query)}`, {
    method,
    headers,
    signal,
    cache,
    keepalive,
    body: body === undefined ? undefined : isForm ? body : JSON.stringify(body),
  });

  const length = response.headers.get("Content-Length");
  if (response.status === 204 || length === "0") {
    return response.ok
      ? { data: undefined as T, response }
      : { error: undefined, response };
  }
  const text = await response.text();
  if (response.ok) {
    return { data: (text ? JSON.parse(text) : undefined) as T, response };
  }
  try {
    return { error: JSON.parse(text), response };
  } catch {
    return { error: text, response };
  }
}

function searchOf(query: ApiOptions["query"]): string {
  const search = new URLSearchParams();
  for (const [name, value] of Object.entries(query ?? {})) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const item of value) search.append(name, String(item));
    } else {
      search.append(name, String(value));
    }
  }
  const encoded = search.toString();
  return encoded ? `?${encoded}` : "";
}
