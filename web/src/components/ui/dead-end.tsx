import type { ReactNode } from "react";
import { BrandMark } from "@/components/brand/BrandMark";
import { Shell } from "@/components/layout/Shell";

export function DeadEnd({
  children,
  heading,
  line,
  note,
}: {
  children: ReactNode;
  heading: string;
  line: string;
  note?: ReactNode;
}) {
  return (
    <Shell className="flex min-h-[58svh] flex-col justify-center py-section">
      <div className="max-w-[46rem]">
        <span className="inline-flex opacity-70">
          <BrandMark size={40} tone="accent" />
        </span>
        <h1 className="mt-6 font-display text-display leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          {heading}
        </h1>
        <p className="mt-5 max-w-[52ch] font-prose text-lede text-mute">
          {line}
        </p>
        <div className="mt-9 flex flex-wrap items-center gap-3">{children}</div>
        {note ? (
          <p className="mt-10 font-mono text-meta text-mute">{note}</p>
        ) : null}
      </div>
    </Shell>
  );
}
