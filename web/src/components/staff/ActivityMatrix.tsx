"use client";

import { ArrowDownRight, ArrowUpRight, Minus, RotateCw } from "lucide-react";
import { type KeyboardEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  SideScroll,
  Slab,
  SlabFoot,
  SlabHead,
  SlabNote,
  SlabTitle,
} from "@/components/ui/slab";
import type { Report } from "@/lib/api/staff";
import { cn } from "@/lib/cn";
import {
  changeOf,
  count,
  isWeekend,
  levelOf,
  reportDate,
  type Series,
  seriesOf,
  weekday,
} from "@/lib/report";

const CELL = "min-w-[0.8rem] flex-1";

const SHADES = [
  "bg-deep/70",
  "bg-accent/25",
  "bg-accent/50",
  "bg-accent/75",
  "bg-accent",
];

/** Every series against every day as one grid, so a day that moved one series is read against the rest. */
export function ActivityMatrix({
  refreshing,
  onRefresh,
  report,
}: {
  refreshing: boolean;
  onRefresh: () => void;
  report: Report;
}) {
  const series = seriesOf(report);
  const days = report.days.map((entry) => entry.day);
  const last = days.length - 1;
  const [held, setHeld] = useState<number | null>(null);
  const shown = held ?? last;

  function move(event: KeyboardEvent<HTMLDivElement>) {
    const step =
      event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
    if (step === 0 && event.key !== "Home" && event.key !== "End") return;
    event.preventDefault();
    if (event.key === "Home") return setHeld(0);
    if (event.key === "End") return setHeld(last);
    setHeld(Math.min(Math.max(shown + step, 0), last));
  }

  return (
    <Slab>
      <SlabHead className="items-center">
        <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
          <SlabTitle>Thirty days</SlabTitle>
          <SlabNote>
            {reportDate(report.from)} to {reportDate(report.through, true)}, in
            UTC
          </SlabNote>
        </div>
        <Button
          className="-my-1 text-mute"
          loading={refreshing}
          onClick={onRefresh}
          size="compact"
          variant="ghost"
        >
          {refreshing ? null : <RotateCw aria-hidden="true" />}
          Refresh
        </Button>
      </SlabHead>

      <div className="flex flex-col gap-4 px-4 py-4 lg:flex-row lg:gap-6">
        <DayReadout day={days[shown]} series={series} shown={shown} />
        <div
          aria-label="The day being read"
          aria-valuemax={last}
          aria-valuemin={0}
          aria-valuenow={shown}
          aria-valuetext={`${weekday(days[shown])} ${reportDate(days[shown], true)}`}
          className="min-w-0 flex-1 rounded-control outline-offset-4"
          onKeyDown={move}
          onPointerLeave={() => setHeld(null)}
          role="slider"
          tabIndex={0}
        >
          <SideScroll>
            <div className="relative min-w-[34rem]">
              <Bands days={days} shown={shown} />
              <div className="relative flex flex-col gap-2">
                {series.map((one) => (
                  <MatrixRow
                    days={days}
                    key={one.id}
                    onHold={setHeld}
                    series={one}
                    shown={shown}
                  />
                ))}
                <div
                  aria-hidden="true"
                  className="flex gap-[3px] pt-1 font-ui text-label text-mute tabular-nums"
                >
                  {days.map((day, index) => (
                    <span className={cn(CELL, "text-center")} key={day}>
                      {index % 5 === 0 || index === last
                        ? new Date(`${day}T00:00:00Z`).getUTCDate()
                        : ""}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </SideScroll>
        </div>
      </div>

      <SlabFoot>
        <span>
          Each row is shaded against its own busiest day. Weekends stand on a
          darker ground.
        </span>
        <span className="flex items-center gap-1.5">
          Quiet
          {SHADES.map((shade) => (
            <span
              aria-hidden="true"
              className={cn("size-3 rounded-[3px]", shade)}
              key={shade}
            />
          ))}
          Busy
        </span>
      </SlabFoot>
    </Slab>
  );
}

/** The ground under the rows, where weekends sit darker and the day being read is lit top to bottom. */
function Bands({ days, shown }: { days: string[]; shown: number }) {
  return (
    <div aria-hidden="true" className="absolute inset-0 flex gap-[3px] pb-5">
      {days.map((day, index) => (
        <span
          className={cn(
            CELL,
            "rounded-[4px]",
            isWeekend(day) && "bg-inset",
            index === shown && "bg-accent-wash/80 ring-1 ring-accent/35",
          )}
          key={day}
        />
      ))}
    </div>
  );
}

function DayReadout({
  day,
  series,
  shown,
}: {
  day: string;
  series: Series[];
  shown: number;
}) {
  return (
    <div className="shrink-0 lg:w-[13rem]">
      <p className="font-ui text-label tracking-[0.06em] text-mute uppercase">
        {weekday(day)}
      </p>
      <p className="font-display text-[1.5rem] leading-tight font-medium tracking-[-0.03em] text-ink">
        {reportDate(day, true)}
      </p>
      <dl
        aria-live="polite"
        className="mt-3 grid grid-cols-2 gap-x-5 border-t border-rule pt-2 lg:grid-cols-1"
      >
        {series.map((one) => (
          <div
            className="flex items-baseline justify-between gap-3 border-b border-rule/50 py-1.5 last:border-b-0"
            key={one.id}
          >
            <dt className="font-ui text-meta text-mute">{one.name}</dt>
            <dd className="font-ui text-meta font-medium text-ink tabular-nums">
              {count.format(one.values[shown])}
            </dd>
          </div>
        ))}
      </dl>
      <p className="mt-3 font-ui text-label text-mute">
        Point at a day, or move through them with the arrow keys.
      </p>
    </div>
  );
}

function MatrixRow({
  days,
  onHold,
  series,
  shown,
}: {
  days: string[];
  onHold: (index: number) => void;
  series: Series;
  shown: number;
}) {
  const change = changeOf(series.total, series.previous);
  const Arrow =
    change.trend === "up"
      ? ArrowUpRight
      : change.trend === "down"
        ? ArrowDownRight
        : Minus;

  return (
    <div>
      <div className="flex items-baseline justify-between gap-4 px-0.5 pb-1">
        <p className="flex min-w-0 items-baseline gap-2 font-ui text-meta">
          <span className="font-medium text-ink">{series.name}</span>
          <span className="text-ink tabular-nums">
            {count.format(series.total)}
          </span>
          <span
            className={cn(
              "inline-flex items-center gap-0.5 text-label whitespace-nowrap",
              change.trend === "flat" ? "text-mute" : "text-accent",
            )}
          >
            <Arrow aria-hidden="true" className="size-3 shrink-0" />
            {change.words}
          </span>
          <span className="font-ui text-label whitespace-nowrap text-mute tabular-nums">
            · busiest day {count.format(series.peak)}
          </span>
        </p>
      </div>
      <div
        aria-label={`${series.name}: ${count.format(series.total)} over 30 days, at most ${count.format(series.peak)} in one day`}
        className="flex gap-[3px]"
        role="img"
      >
        {series.values.map((value, index) => (
          <span
            className={cn(
              CELL,
              "h-6 origin-bottom rounded-[3px] motion-safe:animate-cell-in",
              SHADES[levelOf(value, series.peak)],
              index === shown && "ring-1 ring-accent ring-inset",
            )}
            key={days[index]}
            onPointerEnter={() => onHold(index)}
            style={{ animationDelay: `${index * 12}ms` }}
          />
        ))}
      </div>
    </div>
  );
}
