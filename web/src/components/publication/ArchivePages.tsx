import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { archiveSteps } from "@/lib/archive-pages";
import { pageAddress } from "@/lib/blog-paths";

const STEP =
  "flex min-h-11 min-w-11 items-center justify-center gap-1 rounded-control px-3 text-ui";

export function ArchivePages({
  address,
  page,
  pages,
}: {
  address: string;
  page: number;
  pages: number;
}) {
  const steps = archiveSteps(page, pages);
  if (steps.length === 0) return null;
  return (
    <nav
      aria-label="Archive pages"
      className="mt-group flex flex-wrap items-center justify-between gap-3"
    >
      <Edge
        address={address}
        page={page - 1}
        rel="prev"
        show={page > 1}
        side="newer"
      />
      <ol className="flex list-none flex-wrap items-center justify-center gap-1">
        {steps.map((step, index) =>
          step === "gap" ? (
            <li
              // biome-ignore lint/suspicious/noArrayIndexKey: A gap stands for the pages between two numbers and has no identity of its own.
              key={`gap-${index}`}
              className="px-1 text-meta text-mute"
            >
              <span aria-hidden="true">…</span>
              <span className="sr-only">and more pages</span>
            </li>
          ) : (
            <li key={step}>
              <Link
                aria-current={step === page ? "page" : undefined}
                aria-label={`Page ${step}`}
                className={`${STEP} text-mute hover:bg-deep hover:text-ink aria-[current=page]:bg-deep aria-[current=page]:font-medium aria-[current=page]:text-ink`}
                href={pageAddress(address, step)}
              >
                {step}
              </Link>
            </li>
          ),
        )}
      </ol>
      <Edge
        address={address}
        page={page + 1}
        rel="next"
        show={page < pages}
        side="older"
      />
    </nav>
  );
}

function Edge({
  address,
  page,
  rel,
  show,
  side,
}: {
  address: string;
  page: number;
  rel: "prev" | "next";
  show: boolean;
  side: "newer" | "older";
}) {
  const label = side === "newer" ? "Newer" : "Older";
  if (!show) {
    return (
      <span aria-hidden="true" className={`${STEP} text-mute opacity-40`}>
        {side === "newer" ? <ChevronLeft className="size-4" /> : null}
        {label}
        {side === "older" ? <ChevronRight className="size-4" /> : null}
      </span>
    );
  }
  return (
    <Link
      className={`${STEP} font-medium text-ink hover:bg-deep`}
      href={pageAddress(address, page)}
      rel={rel}
    >
      {side === "newer" ? (
        <ChevronLeft aria-hidden="true" className="size-4" />
      ) : null}
      {label} posts
      {side === "older" ? (
        <ChevronRight aria-hidden="true" className="size-4" />
      ) : null}
    </Link>
  );
}
