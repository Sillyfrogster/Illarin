"use client";

import { Columns3, Download, FileDown } from "lucide-react";
import type { ReactNode } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
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

/** FormatMenu groups what the arrow beside the main button offers: the other formats, the owner's original, sends to connected apps, and the comparison. */
export function FormatMenu({
  children,
  downloads,
  hint,
  onCompare,
  onDownload,
  original = null,
  sends,
  workId,
}: {
  children: ReactNode;
  downloads: DownloadFormat[];
  hint?: string;
  onCompare?: () => void;
  onDownload?: () => void;
  original?: OriginalUpload | null;
  sends?: ReactNode;
  workId: string;
}) {
  const hasDownloads = downloads.length > 0 || original !== null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>{children}</DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72">
        {hasDownloads ? (
          <DropdownMenuGroup>
            <DropdownMenuLabel>{hint ?? "Download"}</DropdownMenuLabel>
            {downloads.map((offered) => (
              <DropdownMenuItem asChild key={offered.format}>
                <a
                  href={downloadAddress({ workId, format: offered.format })}
                  onClick={onDownload}
                >
                  <Download aria-hidden="true" />
                  <span className="min-w-0 flex-1 truncate">
                    {offered.label}
                  </span>
                  {offered.recommended ? (
                    <span className="text-meta text-mute">Recommended</span>
                  ) : null}
                </a>
              </DropdownMenuItem>
            ))}
            {original ? (
              <DropdownMenuItem
                asChild
                className="items-start py-2 [&_svg]:mt-0.5"
              >
                <a href={`/download/${workId}`}>
                  <FileDown aria-hidden="true" />
                  <span className="min-w-0 flex-1">
                    Original upload
                    <span className="block text-meta text-mute">
                      {fileWord(original.mediaType)}, uploaded{" "}
                      {readableDate(original.arrivedAt)}. Edits made since are
                      not in it.
                    </span>
                  </span>
                </a>
              </DropdownMenuItem>
            ) : null}
          </DropdownMenuGroup>
        ) : null}
        {sends ? (
          <>
            {hasDownloads ? <DropdownMenuSeparator /> : null}
            <DropdownMenuGroup>
              <DropdownMenuLabel>Send</DropdownMenuLabel>
              {sends}
            </DropdownMenuGroup>
          </>
        ) : null}
        {onCompare ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onCompare}>
              <Columns3 aria-hidden="true" />
              Compare formats
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
