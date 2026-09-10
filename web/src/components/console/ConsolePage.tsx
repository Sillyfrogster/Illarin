import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";

/** The frame every administration page shares, with what it is above the work itself. */
export function ConsolePage({
  children,
  heading,
  hint,
}: {
  children: ReactNode;
  heading: string;
  hint: string;
}) {
  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          {heading}
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">{hint}</p>
      </header>
      <div className="mt-10 min-w-0">{children}</div>
    </Shell>
  );
}
