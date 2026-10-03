import { cookies } from "next/headers";
import Link from "next/link";
import { Suspense } from "react";
import { BrowseRetry } from "@/components/browse/BrowseRetry";
import { Message } from "@/components/browse/BrowseStates";
import { HomeHero } from "@/components/home/HomeHero";
import { HOME_TYPES, homeRows } from "@/components/home/rows";
import { TypeRow } from "@/components/home/TypeRow";
import { shellClasses } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchWorks } from "@/lib/api/query";
import { cn } from "@/lib/cn";

const ROW_LENGTH = 12;

export default function HomePage() {
  return (
    <>
      <HomeHero />
      <div
        className={cn(
          shellClasses,
          "-mt-32 flex flex-col gap-group max-sm:-mt-20",
        )}
      >
        <Suspense fallback={<RowsLoading />}>
          <Rows />
        </Suspense>
      </div>
    </>
  );
}

async function Rows() {
  const cookie = (await cookies()).toString();
  const home = homeRows(
    await Promise.all(
      HOME_TYPES.map(async (type) => ({
        type,
        page: await fetchWorks(
          { type, sort: "downloads", limit: ROW_LENGTH },
          cookie,
          AbortSignal.timeout(6000),
        ).catch(() => null),
      })),
    ),
  );

  if (home.state === "failed")
    return (
      <Message
        action={<BrowseRetry />}
        body="Try again, or go to Browse."
        title="Recent work could not load."
      />
    );
  if (home.state === "empty")
    return (
      <Message
        action={<Go href="/upload" label="Publish" primary />}
        body="Upload a character or a lorebook and it shows up here."
        title="Nothing published yet"
      />
    );
  if (home.state === "hidden")
    return (
      <Message
        action={<Go href="/browse" label="Browse" />}
        body="Everything here is adult work, and you hide adult work. Change that in Browse."
        title="Adult works are hidden"
      />
    );
  if (home.state === "few")
    return (
      <div className="flex justify-center">
        <Go href="/browse" label="Browse" primary />
      </div>
    );
  return (
    <>
      {home.rows.map((row) => (
        <TypeRow key={row.type} row={row} />
      ))}
      {home.someHidden ? (
        <p className="text-meta text-mute">Some adult works are hidden.</p>
      ) : null}
    </>
  );
}

function Go({
  href,
  label,
  primary = false,
}: {
  href: string;
  label: string;
  primary?: boolean;
}) {
  return (
    <Button asChild variant={primary ? "primary" : undefined}>
      <Link href={href}>{label}</Link>
    </Button>
  );
}

function RowsLoading() {
  return (
    <output
      aria-label="Loading works"
      className="flex gap-4 overflow-hidden py-3"
    >
      {[0, 1, 2, 3, 4, 5].map((slot) => (
        <Skeleton
          className="aspect-3/4 w-[44vw] shrink-0 rounded-card sm:w-[calc((100%-3*1rem)/4)] lg:w-[calc((100%-5*1rem)/6)]"
          key={slot}
        />
      ))}
    </output>
  );
}
