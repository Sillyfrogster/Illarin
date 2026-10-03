"use client";

import { useQuery } from "@tanstack/react-query";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
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
      <Card className="max-w-[40rem] gap-4 p-5">
        <Alert tone="stop">{query.error.message}</Alert>
        <Button
          className="self-start"
          loading={query.isFetching}
          onClick={() => query.refetch()}
          variant="secondary"
        >
          Try again
        </Button>
      </Card>
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
      <Skeleton className="h-[19rem] rounded-card" />
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <Skeleton className="h-72 rounded-card" />
        <Skeleton className="h-72 rounded-card" />
      </div>
    </div>
  );
}
