import createClient from "openapi-fetch";
import { markBrowserMutation } from "./browser-mutation";
import type { paths } from "./schema";

const baseUrl =
  typeof window === "undefined"
    ? (process.env.API_URL ?? "http://localhost:8080")
    : "/api";

const callFetch = (request: Request) => globalThis.fetch(request);

export const api = createClient<paths>({ baseUrl, fetch: callFetch });

api.use({
  onRequest({ request }) {
    markBrowserMutation(request.method, request.headers);
    return request;
  },
});

export type Asset =
  paths["/v1/assets"]["get"]["responses"]["200"]["content"]["application/json"]["items"][number];
