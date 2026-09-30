import type { ReactNode } from "react";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";

export const GRID =
  "m-0 grid list-none grid-cols-2 items-start gap-x-4 gap-y-9 p-0 sm:grid-cols-3 sm:gap-x-5 lg:grid-cols-4 xl:grid-cols-5";

export const CONTROL =
  "inline-flex h-11 min-w-0 items-center gap-1 rounded-control bg-deep px-3 font-ui text-ui font-medium text-ink outline-offset-2 transition-colors duration-200 hover:bg-accent-wash disabled:opacity-55 data-[state=open]:bg-accent-wash motion-reduce:transition-none";

/** Ten placeholder cards while a listing loads. */
export function BrowseLoading() {
  return (
    <output aria-label="Loading works" className="block">
      <div className={GRID}>
        {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((slot) => (
          <div key={slot}>
            <Skeleton className="aspect-5/6 rounded-plate" />
            <Skeleton className="mt-4 h-4 w-3/4" />
            <Skeleton className="mt-2 h-3 w-1/2" />
          </div>
        ))}
      </div>
    </output>
  );
}

/** Message is the empty state for a listing with nothing to show. */
export function Message({
  action,
  body,
  title,
}: {
  action?: ReactNode;
  body?: string;
  title: string;
}) {
  return (
    <Empty className="bg-deep">
      <EmptyHeader>
        <EmptyTitle as="h3">{title}</EmptyTitle>
        {body ? <EmptyDescription>{body}</EmptyDescription> : null}
      </EmptyHeader>
      {action ? <EmptyContent>{action}</EmptyContent> : null}
    </Empty>
  );
}
