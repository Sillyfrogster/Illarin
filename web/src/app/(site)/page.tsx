import { cookies } from "next/headers";
import { Suspense } from "react";
import {
  CatalogChapter,
  CatalogLoading,
} from "@/components/landing/CatalogChapter";
import { HostedLanding } from "@/components/landing/HostedLanding";
import { fetchAssets } from "@/lib/api/query";

export default function LandingPage() {
  return (
    <HostedLanding>
      <Suspense fallback={<CatalogLoading />}>
        <RecentCreations />
      </Suspense>
    </HostedLanding>
  );
}

async function RecentCreations() {
  const cookie = (await cookies()).toString();
  const latest = await fetchAssets(
    { limit: 5 },
    cookie,
    AbortSignal.timeout(6000),
  ).catch(() => null);

  return <CatalogChapter page={latest} />;
}
