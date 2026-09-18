import { afterEach, describe, expect, test } from "bun:test";
import { BROWSER_MUTATION_HEADER } from "./browser-mutation";
import { api } from "./client";

const originalFetch = globalThis.fetch;

function answerWith(response: Response) {
  const sent: Request[] = [];
  Object.defineProperty(globalThis, "fetch", {
    configurable: true,
    writable: true,
    value: async (input: RequestInfo | URL, init?: RequestInit) => {
      sent.push(new Request(input, init));
      return response;
    },
  });
  return sent;
}

afterEach(() => {
  Object.defineProperty(globalThis, "fetch", {
    configurable: true,
    writable: true,
    value: originalFetch,
  });
});

describe("api", () => {
  test("repeats list values in the query and leaves out missing ones", async () => {
    const sent = answerWith(Response.json({ items: [] }));

    await api("GET", "/v1/works", {
      query: { facet: ["a", "b"], type: undefined, limit: 24, q: "two words" },
    });

    const address = new URL(sent[0].url);
    expect(address.pathname).toBe("/v1/works");
    expect(address.searchParams.getAll("facet")).toEqual(["a", "b"]);
    expect(address.searchParams.has("type")).toBe(false);
    expect(address.searchParams.get("limit")).toBe("24");
    expect(address.searchParams.get("q")).toBe("two words");
  });

  test("sends a body as JSON and marks the request as a browser mutation", async () => {
    const sent = answerWith(Response.json({ id: "one" }, { status: 201 }));

    const { data } = await api<{ id: string }>("POST", "/v1/things", {
      body: { name: "one" },
    });

    expect(data).toEqual({ id: "one" });
    expect(sent[0].headers.get("Content-Type")).toBe("application/json");
    expect(sent[0].headers.get(BROWSER_MUTATION_HEADER)).toBe("1");
    expect(await sent[0].json()).toEqual({ name: "one" });
  });

  test("leaves form data for fetch to encode", async () => {
    const sent = answerWith(new Response(null, { status: 204 }));
    const body = new FormData();
    body.append("file", new Blob(["x"]), "x.txt");

    const { data, error, response } = await api<void>("PUT", "/v1/file", {
      body,
    });

    expect(response.status).toBe(204);
    expect(data).toBeUndefined();
    expect(error).toBeUndefined();
    expect(sent[0].headers.get("Content-Type")).toStartWith(
      "multipart/form-data",
    );
  });

  test("returns a refusal body parsed, or as text when it is not JSON", async () => {
    answerWith(Response.json({ error: "No." }, { status: 409 }));
    expect((await api("GET", "/v1/refused")).error).toEqual({ error: "No." });

    answerWith(new Response("<html>too large</html>", { status: 413 }));
    expect((await api("GET", "/v1/refused")).error).toBe(
      "<html>too large</html>",
    );
  });
});
