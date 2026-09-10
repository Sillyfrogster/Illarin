"use client";

import Image from "next/image";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import type { ProfileDistinction } from "@/lib/api/query";

function Mark({ one, size }: { one: ProfileDistinction; size: number }) {
  if (!one.mark) return null;
  return (
    <Image
      alt=""
      className="shrink-0"
      height={size}
      src={one.mark.url}
      style={{ width: size, height: size }}
      unoptimized
      width={size}
    />
  );
}

export function ProfileRecognition({
  badges,
  titles,
}: {
  badges: ProfileDistinction[];
  titles: ProfileDistinction[];
}) {
  const marked = badges.filter((one) => one.mark);
  const spoken = [...badges.filter((one) => !one.mark), ...titles];
  if (marked.length === 0 && spoken.length === 0) return null;

  return (
    <MorphingDisclosure
      lead={
        <ul className="flex list-none flex-wrap items-center gap-x-6 gap-y-3 p-0">
          {marked.length > 0 ? (
            <li className="flex flex-wrap items-center gap-2.5">
              {marked.map((one) => (
                <span key={one.id} title={one.name}>
                  <Mark one={one} size={34} />
                  <span className="sr-only">{one.name}</span>
                </span>
              ))}
            </li>
          ) : null}
          {spoken.map((one) => (
            <li className="font-ui text-ui font-medium text-ink" key={one.id}>
              {one.name}
            </li>
          ))}
        </ul>
      }
      summary="Given by Illarin"
    >
      <dl className="mt-5 grid gap-x-10 gap-y-5 sm:grid-cols-2 xl:grid-cols-3">
        {[...marked, ...spoken].map((one) => (
          <div className="flex items-start gap-3" key={one.id}>
            <Mark one={one} size={30} />
            <div className="min-w-0">
              <dt className="font-ui text-ui font-medium text-ink">
                {one.name}
              </dt>
              {one.explanation ? (
                <dd className="mt-1 font-prose text-meta text-mute">
                  {one.explanation}
                </dd>
              ) : null}
            </div>
          </div>
        ))}
      </dl>
    </MorphingDisclosure>
  );
}
