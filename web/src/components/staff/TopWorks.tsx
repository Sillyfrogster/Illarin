import Image from "next/image";
import Link from "next/link";
import { TypeMark } from "@/components/browse/TypeMark";
import {
  Slab,
  SlabFoot,
  SlabHead,
  SlabNote,
  SlabTitle,
} from "@/components/ui/slab";
import type { BrowseType } from "@/lib/api/query";
import type { ReportWork } from "@/lib/api/staff";
import { count } from "@/lib/report";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";

/** The works handed over most in the report's days, each read against the one above it. */
export function TopWorks({ works }: { works: ReportWork[] }) {
  const most = works[0]?.downloads ?? 0;
  const between = works.reduce((sum, work) => sum + work.downloads, 0);

  return (
    <Slab>
      <SlabHead>
        <SlabTitle>Most downloaded</SlabTitle>
        <SlabNote>Every format counted together</SlabNote>
      </SlabHead>
      {works.length === 0 ? (
        <p className="px-4 py-8 font-ui text-meta text-mute">
          Nothing has been downloaded in the last 30 days.
        </p>
      ) : (
        <ol className="flex list-none flex-col">
          {works.map((work, index) => (
            <li
              className="group relative flex items-center gap-3 border-b border-rule/70 px-4 py-2.5 last:border-b-0 hover:bg-inset/70"
              key={work.id}
            >
              <span className="w-4 shrink-0 text-right font-ui text-label text-mute tabular-nums">
                {index + 1}
              </span>
              <Cover work={work} />
              <div className="min-w-0 flex-1">
                <Link
                  className="block truncate font-ui text-meta font-medium text-ink outline-offset-2 after:absolute after:inset-0 after:content-['']"
                  href={workHref(work.id, work.name)}
                >
                  {work.name}
                </Link>
                <span className="mt-1 flex items-center gap-2">
                  <span className="h-1 w-full max-w-[14rem] rounded-full bg-deep">
                    <span
                      className="block h-full rounded-full bg-accent"
                      style={{ width: `${(work.downloads / most) * 100}%` }}
                    />
                  </span>
                  <span className="font-ui text-label text-mute">
                    {TYPE_LABELS[work.type as BrowseType] ?? work.type}
                  </span>
                </span>
              </div>
              <span className="shrink-0 font-ui text-meta text-ink tabular-nums">
                {count.format(work.downloads)}
              </span>
            </li>
          ))}
        </ol>
      )}
      <SlabFoot>
        <span>
          {works.length === 0
            ? "The list fills as works are handed over."
            : `${count.format(between)} downloads between them`}
        </span>
      </SlabFoot>
    </Slab>
  );
}

function Cover({ work }: { work: ReportWork }) {
  if (work.cover) {
    return (
      <Image
        alt=""
        className="size-8 shrink-0 rounded-[5px] bg-inset object-cover"
        height={64}
        src={work.cover}
        unoptimized
        width={64}
      />
    );
  }
  return (
    <span className="flex size-8 shrink-0 items-center justify-center rounded-[5px] bg-deep">
      <TypeMark className="size-4 text-accent" type={work.type as BrowseType} />
    </span>
  );
}
