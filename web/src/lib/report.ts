import type { Report, ReportDay } from "./api/staff";

export type SeriesId = keyof Omit<ReportDay, "day">;

export type Series = {
  id: SeriesId;
  name: string;
  total: number;
  previous: number;
  peak: number;
  values: number[];
};

/** The five series in the order the report reads them. */
export const SERIES: { id: SeriesId; name: string }[] = [
  { id: "visits", name: "Visits" },
  { id: "downloads", name: "Downloads" },
  { id: "sends", name: "Sends" },
  { id: "signUps", name: "Sign-ups" },
  { id: "publishes", name: "Publishes" },
];

export function seriesOf(report: Report): Series[] {
  return SERIES.map(({ id, name }) => {
    const values = report.days.map((day) => day[id]);
    return {
      id,
      name,
      values,
      total: values.reduce((sum, value) => sum + value, 0),
      previous: report.previous[id],
      peak: Math.max(0, ...values),
    };
  });
}

export type Change = { trend: "up" | "down" | "flat"; words: string };

/** How a total moved against the 30 days before it, as a person reads it. */
export function changeOf(total: number, previous: number): Change {
  if (total === previous) return { trend: "flat", words: "no change" };
  if (previous === 0) return { trend: "up", words: "from none" };
  const percent = Math.round(((total - previous) / previous) * 100);
  return {
    trend: total > previous ? "up" : "down",
    words: `${percent > 0 ? "+" : "−"}${Math.abs(percent)}% on the 30 before`,
  };
}

/** A report day as a person reads it, on the UTC day the totals were counted in. */
export function reportDate(day: string, year = false): string {
  return new Date(`${day}T00:00:00Z`).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    timeZone: "UTC",
    ...(year ? { year: "numeric" } : {}),
  });
}

export type DayNote = { day: string; count: number };

/** The busiest and quietest day of a series, and the weekday it does best on. */
export function shapeOf(series: Series, days: string[]) {
  const busiest = series.values.reduce(
    (best, count, index) =>
      count > best.count ? { day: days[index], count } : best,
    { day: days[0], count: series.values[0] ?? 0 },
  );
  const quietest = series.values.reduce(
    (worst, count, index) =>
      count < worst.count ? { day: days[index], count } : worst,
    { day: days[0], count: series.values[0] ?? 0 },
  );
  const byWeekday = new Map<string, DayNote>();
  for (const [index, count] of series.values.entries()) {
    const name = weekday(days[index]);
    const running = byWeekday.get(name) ?? { day: name, count: 0 };
    byWeekday.set(name, { day: name, count: running.count + count });
  }
  const best = [...byWeekday.values()].reduce(
    (top, one) => (one.count > top.count ? one : top),
    { day: "", count: -1 },
  );
  return { busiest, quietest, weekday: best };
}

export function weekday(day: string): string {
  return new Date(`${day}T00:00:00Z`).toLocaleDateString("en-GB", {
    weekday: "short",
    timeZone: "UTC",
  });
}

export const count = new Intl.NumberFormat("en-GB");

/** Five steps of one hue, from nothing to four shares of the series' own busiest day. */
export const LEVELS = 4;

export function levelOf(value: number, peak: number): number {
  if (value <= 0 || peak <= 0) return 0;
  return Math.max(1, Math.ceil((value / peak) * LEVELS));
}

export function isWeekend(day: string): boolean {
  const at = new Date(`${day}T00:00:00Z`).getUTCDay();
  return at === 0 || at === 6;
}
