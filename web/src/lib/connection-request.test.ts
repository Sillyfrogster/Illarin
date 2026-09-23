import { expect, test } from "bun:test";
import {
  isConnectionRedirect,
  isPendingCodeConnection,
  isPendingConnection,
  isSafeLoopbackRedirect,
} from "./connection-request";

const pending = {
  acceptedFormats: ["chub"],
  appName: "Rookery",
  appVersion: "2.1.0",
  capabilities: ["import"],
  expiresAt: "2026-09-10T12:00:00Z",
  name: "Rookery on the study desk",
  protocolVersion: 1,
  permissions: ["work:receive"],
};

test("reads a pending connection the API returned in full", () => {
  expect(isPendingConnection(pending)).toBe(true);
});

test("refuses a connection missing the name the reader decides about", () => {
  const { appName, ...rest } = pending;

  expect(isPendingConnection(rest)).toBe(false);
});

test("accepts a connection that reports no app version", () => {
  expect(isPendingConnection({ ...pending, appVersion: null })).toBe(true);
});

test("refuses a connection whose permissions are not all names", () => {
  expect(
    isPendingConnection({ ...pending, permissions: ["work:receive", 7] }),
  ).toBe(false);
});

test("a code connection is a pending connection carrying its approval token", () => {
  expect(isPendingCodeConnection(pending)).toBe(false);
  expect(isPendingCodeConnection({ ...pending, approvalToken: "token" })).toBe(
    true,
  );
});

test("reads the callback address an approved browser connection returns", () => {
  expect(
    isConnectionRedirect({ redirectUrl: "http://127.0.0.1:8080/done" }),
  ).toBe(true);
  expect(isConnectionRedirect({ redirectUrl: 12 })).toBe(false);
  expect(isConnectionRedirect(null)).toBe(false);
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
