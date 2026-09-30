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

/** Ten placeholder cards while a listing loads. */
export function BrowseLoading() {
  return (
    <output aria-label="Loading works" className="block">
      <div className={GRID}>
        {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((slot) => (
          <div key={slot}>
            <Skeleton className="aspect-3/4 rounded-plate" />
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
