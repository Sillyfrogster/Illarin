import { describe, expect, test } from "bun:test";
import {
  atLeastAnHourAhead,
  howSoon,
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

describe("howSoon", () => {
  const now = new Date("2026-09-05T09:00:00Z");

  test("counts the days when the instant is more than a day off", () => {
    expect(howSoon("2026-09-12T09:00:00Z", now)).toBe("in 7 days");
    expect(howSoon("2026-09-06T21:00:00Z", now)).toBe("in 1 day");
  });

  test("counts the hours and minutes when it is closer", () => {
    expect(howSoon("2026-09-05T14:00:00Z", now)).toBe("in 5 hours");
    expect(howSoon("2026-09-05T10:00:00Z", now)).toBe("in 1 hour");
    expect(howSoon("2026-09-05T09:20:00Z", now)).toBe("in 20 minutes");
    expect(howSoon("2026-09-05T09:00:30Z", now)).toBe("in under a minute");
  });

  test("says nothing about an instant that has passed", () => {
    expect(howSoon("2026-09-05T08:00:00Z", now)).toBe("");
    expect(howSoon("not-an-instant", now)).toBe("");
  });
});
