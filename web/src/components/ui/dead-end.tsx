import type { ReactNode } from "react";
import { BrandMark } from "@/components/brand/BrandMark";
import { Shell } from "@/components/layout/Shell";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

/** DeadEnd is a whole page that has nothing to show: not found, an error, a closed door. */
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
      <Empty>
        <EmptyHeader>
          <EmptyMedia className="size-auto bg-transparent [&_svg]:size-10">
            <BrandMark size={40} tone="accent" />
          </EmptyMedia>
          <EmptyTitle as="h1" className="text-display leading-[1.05]">
            {heading}
          </EmptyTitle>
          <EmptyDescription className="text-lede">{line}</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>{children}</EmptyContent>
        {note ? <p className="font-mono text-meta text-mute">{note}</p> : null}
      </Empty>
    </Shell>
  );
}
