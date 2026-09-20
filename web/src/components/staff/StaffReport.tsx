"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { Slab } from "@/components/ui/slab";
import { readReport, staffKeys } from "@/lib/api/staff";
import { ActivityMatrix } from "./ActivityMatrix";
import { DayShape } from "./DayShape";
import { ReportTable } from "./ReportTable";
import { TopWorks } from "./TopWorks";

export function StaffReport() {
  const query = useQuery({
    queryKey: staffKeys.report,
    queryFn: ({ signal }) => readReport(signal),
  });
  const report = query.data;

  if (query.isError) {
    return (
      <Slab className="max-w-[40rem] gap-4 p-5">
        <Trouble>{query.error.message}</Trouble>
        <Button
          className="self-start"
          loading={query.isFetching}
          onClick={() => query.refetch()}
          variant="secondary"
        >
          Try again
        </Button>
      </Slab>
    );
  }
  if (!report) return <ReportSkeleton />;

  return (
    <div className="flex flex-col gap-3">
      <ActivityMatrix
        onRefresh={() => query.refetch()}
        refreshing={query.isFetching}
        report={report}
      />
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <TopWorks works={report.topWorks} />
        <DayShape report={report} />
      </div>
      <ReportTable report={report} />
    </div>
  );
}

function ReportSkeleton() {
  return (
    <div aria-hidden="true" className="flex flex-col gap-3">
      <div className="h-[19rem] rounded-plate bg-deep motion-safe:animate-pulse" />
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <div className="h-72 rounded-plate bg-deep motion-safe:animate-pulse" />
        <div className="h-72 rounded-plate bg-deep motion-safe:animate-pulse" />
      </div>
    </div>
  );
}
