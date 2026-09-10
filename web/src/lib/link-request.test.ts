import { expect, test } from "bun:test";
import {
  isLinkRedirect,
  isPendingDeviceLink,
  isPendingLink,
  isSafeLoopbackRedirect,
  refusalMessage,
} from "./link-request";

const pending = {
  acceptedTargets: ["chub"],
  applicationName: "Rookery",
  applicationVersion: "2.1.0",
  capabilities: ["import"],
  expiresAt: "2026-09-10T12:00:00Z",
  instanceName: "Rookery on the study desk",
  protocolVersion: 1,
  scopes: ["asset:receive"],
};

test("reads a pending link the API returned in full", () => {
  expect(isPendingLink(pending)).toBe(true);
});

test("refuses a link missing the name the reader decides about", () => {
  const { applicationName, ...rest } = pending;

  expect(isPendingLink(rest)).toBe(false);
});

test("accepts a link that declares no version", () => {
  expect(isPendingLink({ ...pending, applicationVersion: null })).toBe(true);
});

test("refuses a link whose scopes are not all names", () => {
  expect(isPendingLink({ ...pending, scopes: ["asset:receive", 7] })).toBe(
    false,
  );
});

test("a device link is a pending link carrying its approval token", () => {
  expect(isPendingDeviceLink(pending)).toBe(false);
  expect(isPendingDeviceLink({ ...pending, approvalToken: "token" })).toBe(
    true,
  );
});

test("reads the callback address an approved browser link returns", () => {
  expect(isLinkRedirect({ redirectUrl: "http://127.0.0.1:8080/done" })).toBe(
    true,
  );
  expect(isLinkRedirect({ redirectUrl: 12 })).toBe(false);
  expect(isLinkRedirect(null)).toBe(false);
});

test("opens a callback on the reader's own machine", () => {
  expect(isSafeLoopbackRedirect("http://127.0.0.1:8080/done?code=abc")).toBe(
    true,
  );
  expect(isSafeLoopbackRedirect("http://[::1]:49152/callback")).toBe(true);
});

test("refuses a callback that leaves the reader's own machine", () => {
  expect(isSafeLoopbackRedirect("http://example.com:8080/done")).toBe(false);
  expect(isSafeLoopbackRedirect("https://127.0.0.1:8080/done")).toBe(false);
  expect(isSafeLoopbackRedirect("http://127.0.0.1.example.com/done")).toBe(
    false,
  );
});

test("refuses a callback carrying credentials, a fragment or no port", () => {
  expect(isSafeLoopbackRedirect("http://user:pass@127.0.0.1:8080/done")).toBe(
    false,
  );
  expect(isSafeLoopbackRedirect("http://127.0.0.1:8080/done#token")).toBe(
    false,
  );
  expect(isSafeLoopbackRedirect("http://127.0.0.1/done")).toBe(false);
  expect(isSafeLoopbackRedirect("http://127.0.0.1:99999/done")).toBe(false);
});

test("refuses anything that is not a callback address at all", () => {
  expect(isSafeLoopbackRedirect("javascript:alert(1)")).toBe(false);
  expect(isSafeLoopbackRedirect("")).toBe(false);
});

test("repeats the reason Illarin gave for a refusal", () => {
  expect(refusalMessage({ error: "That code has expired." }, "fallback")).toBe(
    "That code has expired.",
  );
});

test("falls back when Illarin gave no reason", () => {
  expect(refusalMessage({ error: "  " }, "fallback")).toBe("fallback");
  expect(refusalMessage({}, "fallback")).toBe("fallback");
  expect(refusalMessage(null, "fallback")).toBe("fallback");
});
