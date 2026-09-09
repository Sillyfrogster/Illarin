import { ArrowRight } from "lucide-react";
import Link from "next/link";
import type { RecordedVersion } from "@/lib/api/query";
import { versionDate, versionSummary, versionTitle } from "@/lib/asset-updates";
import { assetHistoryHref } from "@/lib/asset-url";

/** What readers have, and the way into everything Illarin recorded before it. */
export function LatestUpdate({
  assetId,
  kind,
  version,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
}) {
  return (
    <section
      aria-labelledby="latest-update"
      className="mt-8 max-w-[42ch] border-rule border-t pt-5"
    >
      <p className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-meta text-mute">
        <span className="font-medium text-ink" id="latest-update">
          {versionTitle(version)}
        </span>
        <time dateTime={version.recordedAt}>{versionDate(version)}</time>
        {version.versionLabel ? (
          <span>Creator's version {version.versionLabel}</span>
        ) : null}
      </p>
      <p className="mt-2 text-meta text-mute">
        {versionSummary(version, kind)}
      </p>
      <Link
        className="mt-2 inline-flex min-h-11 items-center gap-2 text-meta font-medium text-accent hover:text-ink"
        href={assetHistoryHref(assetId)}
      >
        Update history
        <ArrowRight aria-hidden="true" className="size-4" />
      </Link>
    </section>
  );
}
