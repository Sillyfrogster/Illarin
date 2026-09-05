import { describe, expect, test } from "bun:test";
import {
  atLeastAnHourAhead,
  localParts,
  toInstant,
  zoneLabel,
} from "./schedule-time";

describe("toInstant", () => {
  test("turns a local date and time into an instant carrying its offset", () => {
    const instant = toInstant("2026-09-09", "09:00");

    expect(instant).not.toBe("");
    expect(new Date(instant).getHours()).toBe(9);
    expect(/[+-]\d{2}:\d{2}$|Z$/.test(instant)).toBe(true);
  });

  test("answers nothing when either half is missing or unreadable", () => {
    expect(toInstant("", "09:00")).toBe("");
    expect(toInstant("2026-09-09", "")).toBe("");
    expect(toInstant("not-a-date", "09:00")).toBe("");
  });
});

describe("localParts", () => {
  test("splits an instant into the date and time an input shows", () => {
    const instant = toInstant("2026-09-09", "09:00");

    expect(localParts(instant)).toEqual({ date: "2026-09-09", time: "09:00" });
  });

  test("round-trips through toInstant", () => {
    const parts = localParts(toInstant("2026-12-31", "23:59"));

    expect(toInstant(parts.date, parts.time)).toBe(
      toInstant("2026-12-31", "23:59"),
    );
  });
});

describe("atLeastAnHourAhead", () => {
  test("is an hour past the given moment, on the minute", () => {
    const parts = atLeastAnHourAhead(new Date("2026-09-09T09:12:44"));

    expect(parts.time).toBe("10:12");
    expect(parts.date).toBe("2026-09-09");
  });
});

describe("zoneLabel", () => {
  test("names a zone a person recognises", () => {
    expect(zoneLabel().length).toBeGreaterThan(0);
  });
});
