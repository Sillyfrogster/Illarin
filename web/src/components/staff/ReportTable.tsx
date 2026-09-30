"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardAction, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Report } from "@/lib/api/staff";
import { count, isWeekend, reportDate, SERIES, weekday } from "@/lib/report";

export function ReportTable({ report }: { report: Report }) {
  const [shown, setShown] = useState(false);

  return (
    <Collapsible asChild onOpenChange={setShown} open={shown}>
      <Card>
        <CardHeader className="items-center">
          <CardTitle>Day by day</CardTitle>
          <CardAction>
            <CollapsibleTrigger asChild>
              <Button
                className="-my-1 text-mute"
                size="compact"
                variant="ghost"
              >
                {shown ? "Hide the numbers" : "Show the numbers"}
              </Button>
            </CollapsibleTrigger>
          </CardAction>
        </CardHeader>
        <CollapsibleContent className="pt-3">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead scope="col">Day</TableHead>
                {SERIES.map((one) => (
                  <TableHead className="text-right" key={one.id} scope="col">
                    {one.name}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {report.days.map((day, index) => (
                <TableRow
                  className={isWeekend(day.day) ? "bg-inset/60" : ""}
                  index={index}
                  key={day.day}
                >
                  <TableHead
                    className="h-auto py-2 font-normal text-ink"
                    scope="row"
                  >
                    {weekday(day.day)} {reportDate(day.day, true)}
                  </TableHead>
                  {SERIES.map((one) => (
                    <TableCell
                      className={
                        day[one.id] === 0
                          ? "text-right text-mute tabular-nums"
                          : "text-right tabular-nums"
                      }
                      key={one.id}
                    >
                      {count.format(day[one.id])}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
