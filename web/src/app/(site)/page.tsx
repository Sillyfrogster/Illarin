import { cookies } from "next/headers";
import { Suspense } from "react";
import {
  BrowseChapter,
  BrowseLoading,
} from "@/components/landing/BrowseChapter";
import { HostedLanding } from "@/components/landing/HostedLanding";
import { fetchWorks } from "@/lib/api/query";

export default function LandingPage() {
  return (
    <HostedLanding>
      <Suspense fallback={<BrowseLoading />}>
        <RecentCreations />
      </Suspense>
    </HostedLanding>
  );
}

async function RecentCreations() {
  const cookie = (await cookies()).toString();
  const latest = await fetchWorks(
    { limit: 5 },
    cookie,
    AbortSignal.timeout(6000),
  ).catch(() => null);

  return <BrowseChapter page={latest} />;
}
