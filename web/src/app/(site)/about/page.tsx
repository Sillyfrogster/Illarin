import { cookies } from "next/headers";
import { Suspense } from "react";
import { BrowseChapter, BrowseLoading } from "@/components/about/BrowseChapter";
import { Journey } from "@/components/about/Journey";
import { fetchWorks } from "@/lib/api/query";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "About",
  "How a character goes from an idea to a work on Illarin.",
);

export default function AboutPage() {
  return (
    <Journey>
      <Suspense fallback={<BrowseLoading />}>
        <RecentCreations />
      </Suspense>
    </Journey>
  );
}

async function RecentCreations() {
  const cookie = (await cookies()).toString();
  const latest = await fetchWorks(
    { limit: 3 },
    cookie,
    AbortSignal.timeout(6000),
  ).catch(() => null);

  return <BrowseChapter page={latest} />;
}
