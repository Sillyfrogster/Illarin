"use client";

import { useEffect, useState } from "react";
import { ChangeList } from "@/components/changes/ChangeList";
import { compareAssetVersions, type VersionComparison } from "@/lib/api/query";

export function VersionChanges({
  assetId,
  kind,
  from,
  to,
}: {
  assetId: string;
  kind: string;
  from: number;
  to: number;
}) {
  const [compared, setCompared] = useState<VersionComparison | null>(null);
  const [refusal, setRefusal] = useState("");
  const [reading, setReading] = useState(true);

  useEffect(() => {
    let current = true;
    setReading(true);
    void compareAssetVersions(assetId, from, to).then((answer) => {
      if (!current) return;
      setCompared(answer.compared);
      setRefusal(answer.compared ? "" : answer.refusal);
      setReading(false);
    });
    return () => {
      current = false;
    };
  }, [assetId, from, to]);

  if (reading) return <ComparisonSkeleton />;
  if (!compared) return <Refusal>{refusal}</Refusal>;
  if (compared.unavailable) return <Refusal>{compared.unavailable}</Refusal>;

  return (
    <div className="mt-4 max-w-[70ch]">
      {compared.promptsWithheld ? (
        <p className="mb-4 max-w-[60ch] text-meta text-mute">
          This {kind} keeps its prompts for linked applications, so their
          wording is not shown here.
        </p>
      ) : null}

      {compared.groups.length === 0 ? (
        <p className="text-meta text-mute">
          Nothing differs between these two versions.
        </p>
      ) : (
        <ChangeList groups={compared.groups} />
      )}
    </div>
  );
}

function Refusal({ children }: { children: string }) {
  return (
    <p className="mt-4 max-w-[60ch] text-meta text-stop" role="alert">
      {children}
    </p>
  );
}

function ComparisonSkeleton() {
  return (
    <div className="mt-4" aria-live="polite">
      <span className="sr-only">Reading the comparison</span>
      <div aria-hidden="true" className="grid gap-2">
        <div className="h-3 w-24 animate-pulse rounded-control bg-deep" />
        <div className="h-10 animate-pulse rounded-control bg-deep" />
        <div className="h-10 w-3/4 animate-pulse rounded-control bg-deep" />
      </div>
    </div>
  );
}
