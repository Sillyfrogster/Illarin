"use client";

import { Check } from "lucide-react";
import Link from "next/link";
import { type RefObject, useEffect, useState } from "react";
import { KindMark } from "@/components/catalog/KindMark";
import { Button } from "@/components/ui/button";
import {
  type BrowseKind,
  fetchPreservedNamespaces,
  type PreservedNamespace,
} from "@/lib/api/query";
import { assetHref } from "@/lib/asset-url";
import type { ImportedAsset } from "@/lib/import-stage";
import { KIND_LABELS } from "@/lib/kinds";
import { describePreservedNamespaces } from "@/lib/preserved";

export function ImportReceipt({
  asset,
  headingRef,
  onBeginAgain,
}: {
  asset: ImportedAsset;
  headingRef: RefObject<HTMLHeadingElement | null>;
  onBeginAgain: () => void;
}) {
  const [preserved, setPreserved] = useState<PreservedNamespace[] | null>(null);
  const kind = asset.kind as BrowseKind;
  const label = (KIND_LABELS[kind] ?? asset.kind).toLowerCase();

  useEffect(() => {
    let active = true;
    void fetchPreservedNamespaces(asset.id).then((found) => {
      if (active) setPreserved(found);
    });
    return () => {
      active = false;
    };
  }, [asset.id]);

  const carried =
    preserved && preserved.length > 0
      ? describePreservedNamespaces(preserved.map(({ name }) => name))
      : null;

  return (
    <section className="mt-10">
      <h2
        className="flex items-center gap-2.5 font-display text-section font-medium text-ink"
        ref={headingRef}
        tabIndex={-1}
      >
        <Check
          aria-hidden="true"
          className="text-accent"
          size={22}
          strokeWidth={2}
        />
        Import complete
      </h2>

      <p className="mt-4 flex items-center gap-2 text-lede text-ink wrap-anywhere">
        <KindMark className="size-5 shrink-0 text-mute" kind={kind} />
        {asset.name}
      </p>
      <p className="mt-1 text-meta text-mute">
        A private {label} draft. Nobody else can open it until you publish it.
      </p>

      <div className="mt-6 rounded-plate bg-deep p-5">
        <h3 className="font-display text-ui font-medium text-ink">
          Imported content
        </h3>
        <p className="mt-2 text-meta text-mute">
          Supported content is ready to edit. Extra file data is preserved for
          compatible downloads.
        </p>
        <p className="mt-3 text-meta text-ink">
          {preserved === null
            ? "Loading import details…"
            : carried
              ? `Your file also carried ${carried}. Illarin keeps those details with the original and sends them back out in compatible downloads.`
              : "All imported content is editable. No extra file data was found."}
        </p>
      </div>

      <div className="mt-6 flex flex-wrap items-center gap-2">
        <Button asChild variant="primary">
          <Link href={assetHref(asset.id, asset.name)}>Open the {label}</Link>
        </Button>
        <Button onClick={onBeginAgain} variant="ghost">
          Import another file
        </Button>
      </div>
    </section>
  );
}
