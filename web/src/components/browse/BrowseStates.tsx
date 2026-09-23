import type { ReactNode } from "react";

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
            <div className="aspect-5/6 animate-pulse rounded-plate bg-deep motion-reduce:animate-none" />
            <div className="mt-4 h-4 w-3/4 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
            <div className="mt-2 h-3 w-1/2 animate-pulse rounded-control bg-deep motion-reduce:animate-none" />
          </div>
        ))}
      </div>
    </output>
  );
}

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
    <div className="rounded-plate bg-deep px-6 py-14 text-center sm:px-12">
      <h3 className="font-display text-title font-medium tracking-[-0.02em]">
        {title}
      </h3>
      {body ? (
        <p className="mx-auto mt-3 max-w-[46ch] text-prose text-mute">{body}</p>
      ) : null}
      {action ? <div className="mt-6 flex justify-center">{action}</div> : null}
    </div>
  );
}
