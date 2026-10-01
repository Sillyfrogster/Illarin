import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { Report } from "@/lib/api/staff";
import { count, reportDate, seriesOf, shapeOf, weekday } from "@/lib/report";

/** What the visits say about the month, from its loudest day to the weekday it lives on. */
export function DayShape({ report }: { report: Report }) {
  const days = report.days.map((day) => day.day);
  const visits = seriesOf(report)[0];
  const shape = shapeOf(visits, days);
  const lines = [
    {
      label: "Busiest day",
      value: `${weekday(shape.busiest.day)} ${reportDate(shape.busiest.day)}`,
      note: `${count.format(shape.busiest.count)} visits`,
    },
    {
      label: "Quietest day",
      value: `${weekday(shape.quietest.day)} ${reportDate(shape.quietest.day)}`,
      note: `${count.format(shape.quietest.count)} visits`,
    },
    {
      label: "Best weekday",
      value: shape.weekday.day,
      note: `${count.format(shape.weekday.count)} visits in 30 days`,
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>The month's shape</CardTitle>
        <CardDescription>By visits</CardDescription>
      </CardHeader>
      <dl className="flex flex-col pt-2">
        {lines.map((line) => (
          <div
            className="flex items-baseline justify-between gap-4 border-b border-rule/70 px-5 py-3 last:border-b-0"
            key={line.label}
          >
            <dt className="font-ui text-meta text-mute">{line.label}</dt>
            <dd className="text-right">
              <span className="block font-ui text-meta font-medium text-ink">
                {line.value}
              </span>
              <span className="block font-ui text-label text-mute tabular-nums">
                {line.note}
              </span>
            </dd>
          </div>
        ))}
      </dl>
    </Card>
  );
}
