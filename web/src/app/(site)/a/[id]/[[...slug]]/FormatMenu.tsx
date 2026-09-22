"use client";

import { Download, FileDown } from "lucide-react";
import type { ReactNode } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { DownloadFormat, OriginalUpload } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import { downloadAddress } from "@/lib/work-send";

function fileWord(mediaType: string): string {
  if (mediaType.startsWith("image/")) {
    return mediaType.slice("image/".length).toUpperCase();
  }
  if (mediaType === "application/json") return "JSON";
  if (mediaType === "application/zip") return "Archive";
  return "File";
}

/** FormatMenu lists the formats a work downloads in, one click each, with the original upload for its owner and any extra items. */
export function FormatMenu({
  children,
  downloads,
  extra,
  original = null,
  version,
  workId,
  onDownload,
}: {
  children: ReactNode;
  downloads: DownloadFormat[];
  extra?: ReactNode;
  original?: OriginalUpload | null;
  version?: number;
  workId: string;
  onDownload?: () => void;
}) {
  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>{children}</DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {downloads.map((offered) => (
          <DropdownMenuItem asChild key={offered.format}>
            <a
              href={downloadAddress({
                workId,
                format: offered.format,
                version,
              })}
              onClick={onDownload}
            >
              <Download aria-hidden="true" />
              <span className="min-w-0">
                {offered.label}
                {offered.recommended && downloads.length > 1 ? (
                  <span className="block text-meta text-mute">Recommended</span>
                ) : null}
              </span>
            </a>
          </DropdownMenuItem>
        ))}
        {original ? (
          <>
            {downloads.length > 0 ? <DropdownMenuSeparator /> : null}
            <DropdownMenuItem asChild>
              <a href={`/download/${workId}`}>
                <FileDown aria-hidden="true" />
                <span className="min-w-0">
                  Original upload
                  <span className="block text-meta text-mute">
                    {original.label ? `${original.label} · ` : ""}
                    {fileWord(original.mediaType)}, uploaded{" "}
                    {readableDate(original.arrivedAt)}. Edits made since are not
                    in it.
                  </span>
                </span>
              </a>
            </DropdownMenuItem>
          </>
        ) : null}
        {extra}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
