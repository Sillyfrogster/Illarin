import { cookies } from "next/headers";
import Link from "next/link";
import { BrowseSurface } from "@/components/browse/BrowseSurface";
import { Shell } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { Gate } from "@/components/ui/gate";
import { api } from "@/lib/api/client";
import { fetchDeletedWorks, fetchWorks } from "@/lib/api/query";
import type { SessionState } from "@/lib/api/shapes";
import { readBrowseFilters } from "@/lib/browse-url";
import { pageMetadata } from "@/lib/site-metadata";
import { DeletedWorks } from "./DeletedWorks";

export const metadata = pageMetadata(
  "Your work",
  "Your drafts and published work.",
);

export default async function YourWork({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const cookie = (await cookies()).toString();
  const { data: session, error } = await api<SessionState>(
    "GET",
    "/v1/auth/session",
    { headers: { cookie }, cache: "no-store" },
  );
  if (error) throw new Error("Could not read your account");
  if (!session?.user)
    return (
      <Shell className="py-12">
        <Gate
          heading="Sign in to see your work"
          line="Your drafts and published work are here."
          action="Sign in"
          href="/sign-in?returnTo=%2Fwork"
        />
      </Shell>
    );
  const params = await searchParams;
  const deleted = params.deleted === "true";
  const filters = readBrowseFilters(params);
  const handle = session.user.handle;
  const items = deleted ? await fetchDeletedWorks(handle, cookie) : null;
  if (deleted && items === null)
    throw new Error("Could not load recently deleted work");
  const initialPage = deleted
    ? null
    : await fetchWorks({ ...filters, creator: handle, limit: 24 }, cookie);
  return (
    <>
      <Shell className="pt-10 pb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <h1 className="font-display text-title font-medium">Your work</h1>
          <Button asChild variant="primary">
            <Link href="/upload">Upload</Link>
          </Button>
        </div>
        <nav aria-label="Your work" className="mt-6 flex flex-wrap gap-2">
          <Button asChild variant={deleted ? "ghost" : "primary"}>
            <Link href="/work" aria-current={!deleted ? "page" : undefined}>
              All work
            </Link>
          </Button>
          <Button asChild variant={deleted ? "primary" : "ghost"}>
            <Link
              href="/work?deleted=true"
              aria-current={deleted ? "page" : undefined}
            >
              Recently deleted
            </Link>
          </Button>
        </nav>
      </Shell>
      {deleted ? (
        <DeletedWorks key={JSON.stringify(items)} initialItems={items ?? []} />
      ) : (
        <BrowseSurface
          basePath="/work"
          creator={handle}
          filters={filters}
          heading="Your work"
          initialPage={initialPage}
          search={{
            label: "Search your work",
            placeholder: "Search your work",
          }}
        />
      )}
    </>
  );
}
