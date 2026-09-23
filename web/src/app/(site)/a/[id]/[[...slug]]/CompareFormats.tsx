"use client";

import { Check, Minus } from "lucide-react";
import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { type FormatComparison, fetchFormatTable } from "@/lib/api/query";
import { cn } from "@/lib/cn";

type Reading =
  | { state: "unread" }
  | { state: "read"; table: FormatComparison }
  | { state: "refused" };

const ROW_HEAD =
  "sticky left-0 z-10 h-auto w-[11rem] min-w-[8rem] bg-plane py-3 align-top whitespace-normal";

const GRADE_WORDS: Record<string, string> = {
  full: "Yes",
  partial: "Partly",
  none: "No",
};

/** CompareFormats opens the comparison of every format this type downloads in, field by field. */
export function CompareFormats({
  type,
  typeLabel,
  open,
  onOpenChange,
}: {
  type: string;
  typeLabel: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [reading, setReading] = useState<Reading>({ state: "unread" });
  useEffect(() => {
    if (!open) return;
    let live = true;
    fetchFormatTable().then(
      (table) => live && setReading({ state: "read", table }),
      () => live && setReading({ state: "refused" }),
    );
    return () => {
      live = false;
    };
  }, [open]);

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-[820px] p-6 sm:p-8">
        <DialogTitle className="pr-10 font-display text-title font-medium text-ink">
          Compare formats
        </DialogTitle>
        <DialogDescription className="mt-2 text-ui text-mute">
          What each format carries of a {typeLabel}, and which apps read it.
        </DialogDescription>
        <div className="mt-6 min-h-0 overflow-y-auto">
          {reading.state === "read" ? (
            <ComparisonTable table={reading.table} type={type} />
          ) : reading.state === "refused" ? (
            <p className="text-meta text-stop">
              Illarin could not load the comparison. Close this and try again.
            </p>
          ) : (
            <p aria-busy="true" className="text-meta text-mute">
              Loading the comparison…
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ComparisonTable({
  table,
  type,
}: {
  table: FormatComparison;
  type: string;
}) {
  const formats = table.formats.filter((column) => column.type === type);
  const fields = formats.find((column) => !column.keepsUpload)?.fields ?? [];
  const appLabel = (id: string) =>
    table.apps.find((app) => app.id === id)?.label ?? id;

  return (
    <Table className="text-ui">
      <TableHeader>
        <TableRow className="hover:bg-transparent">
          <TableHead className={ROW_HEAD}>
            <span className="sr-only">Field</span>
          </TableHead>
          {formats.map((column) => (
            <TableHead
              className="whitespace-normal text-ink"
              key={column.id}
              scope="col"
            >
              {column.label}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow className="hover:bg-transparent">
          <TableHead className={ROW_HEAD} scope="row">
            Read by
          </TableHead>
          {formats.map((column) => (
            <TableCell className="py-3 whitespace-normal" key={column.id}>
              {column.readBy.map(appLabel).join(", ")}
            </TableCell>
          ))}
        </TableRow>
        {formats.some((column) => column.keepsUpload) ? (
          <TableRow className="hover:bg-transparent">
            <TableHead className={ROW_HEAD} scope="row">
              File
            </TableHead>
            {formats.map((column) => (
              <TableCell className="py-3 whitespace-normal" key={column.id}>
                {column.keepsUpload
                  ? "The upload, untouched"
                  : "Written by Illarin"}
              </TableCell>
            ))}
          </TableRow>
        ) : null}
        {fields.map((field, row) => {
          const shared = sharedNote(
            formats.map((column) => column.fields[row]),
          );
          return (
            <TableRow className="hover:bg-transparent" key={field.field}>
              <TableHead className={ROW_HEAD} scope="row">
                {field.label}
                {shared ? (
                  <span className="mt-1 block max-w-[28ch] text-meta font-normal text-mute">
                    {shared}
                  </span>
                ) : null}
              </TableHead>
              {formats.map((column) => (
                <Cell
                  key={column.id}
                  noted={!shared}
                  support={column.fields[row]}
                />
              ))}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

/** sharedNote is the one note every format gives a field, so the row says it once. */
function sharedNote(
  cells: ({ grade: string; note: string } | undefined)[],
): string {
  const note = cells[0]?.note ?? "";
  return note && cells.every((cell) => cell?.note === note) ? note : "";
}

function Cell({
  noted,
  support,
}: {
  noted: boolean;
  support: { grade: string; note: string } | undefined;
}) {
  const grade = support?.grade ?? "none";
  return (
    <TableCell className="min-w-[9rem] py-3 align-top whitespace-normal">
      <span
        className={cn(
          "inline-flex items-center gap-1.5",
          grade === "none" ? "text-mute" : "text-ink",
        )}
      >
        {grade === "none" ? (
          <Minus aria-hidden="true" className="size-3.5" />
        ) : (
          <Check
            aria-hidden="true"
            className={cn("size-3.5", grade === "full" && "text-accent")}
          />
        )}
        {GRADE_WORDS[grade] ?? grade}
      </span>
      {noted && support?.note ? (
        <span className="mt-1 block max-w-[28ch] text-meta text-mute">
          {support.note}
        </span>
      ) : null}
    </TableCell>
  );
}
