import Image from "next/image";
import Link from "next/link";
import { TypeMark } from "@/components/browse/TypeMark";
import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
} from "@/components/ui/timeline";
import type { RecentVersion } from "@/lib/api/query";
import { foldVersions, versionWords } from "@/lib/profile-portfolio";
import { workDisplayName } from "@/lib/work-name";
import { isWorkType } from "@/lib/work-types";
import { workHistoryHref, workHref } from "@/lib/work-url";

const SHOWN = 6;

/** The last few versions across the creator's work, newest first, each opening its work. */
export function RecentVersions({ versions }: { versions: RecentVersion[] }) {
  if (versions.length === 0) return null;
  const folded = foldVersions(versions).slice(0, SHOWN);
  return (
    <section aria-labelledby="recent-versions" className="min-w-0">
      <h2
        className="font-display text-section font-medium tracking-[-0.02em] text-ink"
        id="recent-versions"
      >
        Recent versions
      </h2>
      <Timeline className="mt-5 [--timeline-dot-offset:0.85rem] [--timeline-dot-size:0.6rem]">
        {folded.map(({ version, earlier }) => (
          <TimelineItem
            className="gap-3.5 pb-1 last:pb-0"
            key={`${version.workId}-${version.number}`}
          >
            <TimelineDot className="border-accent bg-accent" />
            <TimelineConnector />
            <TimelineContent asChild>
              <Link
                className="group flex min-w-0 items-start gap-3.5 rounded-control py-1.5 pr-2 -mr-2 text-ink hover:text-ink"
                href={
                  version.initial
                    ? workHref(version.workId, version.workName)
                    : workHistoryHref(
                        version.workId,
                        version.workName,
                        version.number,
                      )
                }
              >
                <span className="block size-12 shrink-0 overflow-hidden rounded-control bg-deep">
                  {version.cover ? (
                    <Image
                      alt=""
                      className="size-full object-cover transition-transform duration-500 group-hover:scale-105 motion-reduce:transform-none motion-reduce:transition-none"
                      height={48}
                      src={version.cover}
                      unoptimized
                      width={48}
                    />
                  ) : (
                    <span className="grid size-full place-items-center">
                      {isWorkType(version.workType) ? (
                        <TypeMark
                          className="size-4 text-accent"
                          type={version.workType}
                        />
                      ) : null}
                    </span>
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-ui text-ui font-medium transition-colors duration-200 group-hover:text-accent motion-reduce:transition-none">
                    {workDisplayName(version.workName)}
                  </span>
                  <span className="block font-ui text-meta text-mute">
                    {versionWords(version)}
                  </span>
                  {version.summary ? (
                    <span className="mt-0.5 line-clamp-2 block font-prose text-meta text-mute [overflow-wrap:anywhere]">
                      {version.summary}
                    </span>
                  ) : null}
                  {earlier > 0 ? (
                    <span className="mt-1 block font-ui text-label text-mute">
                      {earlier === 1
                        ? "1 earlier version"
                        : `${earlier} earlier versions`}
                    </span>
                  ) : null}
                </span>
              </Link>
            </TimelineContent>
          </TimelineItem>
        ))}
      </Timeline>
    </section>
  );
}
