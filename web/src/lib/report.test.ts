import { expect, test } from "bun:test";
import type { Report } from "./api/staff";
import {
  changeOf,
  isWeekend,
  levelOf,
  reportDate,
  seriesOf,
  shapeOf,
} from "./report";

test("each series totals its days, knows its busiest one and how it moved", () => {
  const report: Report = {
    from: "2026-08-21",
    through: "2026-08-23",
    days: [
      {
        day: "2026-08-21",
        visits: 4,
        downloads: 0,
        sends: 0,
        signUps: 0,
        publishes: 1,
      },
      {
        day: "2026-08-22",
        visits: 9,
        downloads: 2,
        sends: 0,
        signUps: 0,
        publishes: 0,
      },
      {
        day: "2026-08-23",
        visits: 1,
        downloads: 5,
        sends: 0,
        signUps: 0,
        publishes: 0,
      },
    ],
    previous: { visits: 7, downloads: 14, sends: 0, signUps: 0, publishes: 0 },
    topWorks: [],
  };

  const [visits, downloads, sends] = seriesOf(report);

  expect(visits).toMatchObject({
    name: "Visits",
    total: 14,
    peak: 9,
    values: [4, 9, 1],
    previous: 7,
  });
  expect(downloads).toMatchObject({ total: 7, peak: 5 });
  expect(sends).toMatchObject({ total: 0, peak: 0, values: [0, 0, 0] });
  expect(changeOf(visits.total, visits.previous)).toEqual({
    trend: "up",
    words: "+100% on the 30 before",
  });
  expect(changeOf(downloads.total, downloads.previous)).toEqual({
    trend: "down",
    words: "−50% on the 30 before",
  });
  expect(changeOf(0, 0)).toEqual({ trend: "flat", words: "no change" });
  expect(changeOf(3, 0)).toEqual({ trend: "up", words: "from none" });
  expect(reportDate("2026-08-21")).toBe("21 Aug");
  expect(reportDate("2026-08-21", true)).toBe("21 Aug 2026");

  expect([0, 1, 3, 9].map((value) => levelOf(value, 9))).toEqual([0, 1, 2, 4]);
  expect(levelOf(5, 0)).toBe(0);
  expect([isWeekend("2026-08-22"), isWeekend("2026-08-24")]).toEqual([
    true,
    false,
  ]);

  const days = report.days.map((day) => day.day);
  expect(shapeOf(visits, days)).toEqual({
    busiest: { day: "2026-08-22", count: 9 },
    quietest: { day: "2026-08-23", count: 1 },
    weekday: { day: "Sat", count: 9 },
  });
});
