"use client";

import { Columns3, Download, FileDown } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import type { DownloadFormat, OriginalUpload } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableDate } from "@/lib/dates";
import { downloadAddress } from "@/lib/work-send";

/** leftOut names what a format does not carry in full, or nothing when it keeps everything. */
export function leftOut(format: DownloadFormat): string {
  const missing = format.roles.filter((role) => role.verdict !== "carried");
  if (missing.length === 0) return "Keeps everything";
  return `Leaves out ${missing.map((role) => role.label.toLowerCase()).join(", ")}`;
}

function amount(role: DownloadFormat["roles"][number]): string {
  if (role.verdict === "dropped") return "Left out";
  if (role.verdict === "reduced") return role.reason ?? "In part";
  return role.sample.count > 1 ? role.sample.count.toLocaleString("en-US") : "";
}

/** FileContents is the back of the work card: what the recommended file holds, the other files, and the other places to send it. */
export function FileContents({
  main,
  others,
  original,
  sends,
  workId,
  onCompare,
  onDownload,
}: {
  main: DownloadFormat | null;
  others: DownloadFormat[];
  original: OriginalUpload | null;
  sends: ReactNode;
  workId: string;
  onCompare: (() => void) | null;
  onDownload: () => void;
}) {
  return (
    <div className="flex flex-col gap-6 pt-1">
      {main ? (
        <section>
          <h2 className="text-ui font-medium text-ink">
            In the {main.label} file
          </h2>
          <ul className="mt-3 flex list-none flex-col">
            {main.roles.map((role) => (
              <li
                className="flex min-h-9 items-baseline justify-between gap-4 border-t border-rule/60 py-2 text-ui first:border-t-0"
                key={role.role}
              >
                <span
                  className={cn(
                    role.verdict === "dropped" ? "text-mute" : "text-ink",
                  )}
                >
                  {role.label}
                </span>
                <span className="text-meta text-mute tabular-nums">
                  {amount(role)}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {others.length > 0 || original ? (
        <section>
          <h2 className="text-meta text-mute">Other files</h2>
          <ul className="mt-2 flex list-none flex-col gap-1">
            {others.map((format) => (
              <li key={format.format}>
                <a
                  className="-mx-2 flex min-h-control items-center gap-3 rounded-control px-2 py-1.5 text-ui text-ink hover:bg-fill"
                  href={downloadAddress({ workId, format: format.format })}
                  onClick={onDownload}
                >
                  <Download
                    aria-hidden="true"
                    className="size-4 shrink-0 text-mute"
                  />
                  <span className="min-w-0 flex-1">
                    {format.label}
                    <span className="block text-meta text-mute">
                      {leftOut(format)}
                    </span>
                  </span>
                </a>
              </li>
            ))}
            {original ? (
              <li>
                <a
                  className="-mx-2 flex min-h-control items-center gap-3 rounded-control px-2 py-1.5 text-ui text-ink hover:bg-fill"
                  href={`/download/${workId}`}
                >
                  <FileDown
                    aria-hidden="true"
                    className="size-4 shrink-0 text-mute"
                  />
                  <span className="min-w-0 flex-1">
                    Original upload
                    <span className="block text-meta text-mute">
                      Uploaded {readableDate(original.arrivedAt)}. Edits made
                      since are not in it.
                    </span>
                  </span>
                </a>
              </li>
            ) : null}
          </ul>
        </section>
      ) : null}

      {sends ? (
        <section>
          <h2 className="text-meta text-mute">Send to your apps</h2>
          <div className="mt-2 flex flex-col items-start gap-1">{sends}</div>
        </section>
      ) : null}

      {onCompare ? (
        <Button
          className="-ml-3 self-start"
          onClick={onCompare}
          size="compact"
          variant="ghost"
        >
          <Columns3 aria-hidden="true" />
          Compare every format
        </Button>
      ) : null}
    </div>
  );
}
