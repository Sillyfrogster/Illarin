import Image from "next/image";
import Link from "next/link";
import { TypeMark } from "@/components/browse/TypeMark";
import { Item, ItemGroup } from "@/components/ui/item";
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
      <ItemGroup as="ol" className="mt-4 -mx-3 bg-transparent">
        {folded.map(({ version, earlier }) => (
          <Item
            className="p-0 not-first:border-t-0"
            key={`${version.workId}-${version.number}`}
          >
            <Link
              className="group flex min-w-0 items-start gap-3.5 rounded-control px-3 py-2 text-ink hover:text-ink"
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
                    className="size-full object-cover"
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
                <span className="block truncate font-ui text-ui font-medium transition-colors duration-80 group-hover:text-accent">
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
          </Item>
        ))}
      </ItemGroup>
    </section>
  );
}
