"use client";

import { RotateCcw } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { KindMark } from "@/components/catalog/KindMark";
import { Shell } from "@/components/layout/Shell";
import { Button } from "@/components/ui/button";
import { type DeletedAsset, restoreAsset } from "@/lib/api/query";
import { assetDisplayName } from "@/lib/asset-name";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import { KIND_LABELS } from "@/lib/kinds";

function restoreDeadline(value: string) {
  return new Date(value).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

/** What the creator deleted, for as long as Illarin can still give it back. */
export function DeletedAssets({
  initialItems,
}: {
  initialItems: DeletedAsset[];
}) {
  const router = useRouter();
  const [items, setItems] = useState(initialItems);
  const [pending, setPending] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  async function restore(item: DeletedAsset) {
    if (pending) return;
    setPending(item.id);
    setMessage("");
    setItems((current) =>
      current.filter((candidate) => candidate.id !== item.id),
    );
    try {
      await restoreAsset(item.id);
      router.refresh();
    } catch {
      setItems((current) => [item, ...current]);
      setMessage(
        `${assetDisplayName(item.name)} could not be restored. Try again.`,
      );
    } finally {
      setPending(null);
    }
  }

  return (
    <Shell
      aria-labelledby="deleted-heading"
      as="section"
      className="scroll-mt-28 pb-chapter"
      id="deleted"
    >
      <div className="border-t border-rule pt-10">
        <h2
          className="font-display text-title font-medium tracking-[-0.02em]"
          id="deleted-heading"
        >
          Deleted
        </h2>
        <p className="mt-2 max-w-[56ch] font-prose text-ui text-mute">
          These creations stay here briefly before their files are cleared.
        </p>

        {message ? (
          <p className="mt-4 font-ui text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}

        {items.length > 0 ? (
          <ul className="mt-6 flex list-none flex-col gap-2 p-0">
            {items.map((item) => (
              <li
                className="flex flex-wrap items-center gap-x-6 gap-y-3 rounded-plate bg-deep px-5 py-4"
                key={item.id}
              >
                <div className="min-w-0 flex-1">
                  <h3 className="font-display text-section font-medium tracking-[-0.02em] [overflow-wrap:anywhere]">
                    {assetDisplayName(item.name)}
                  </h3>
                  <p className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 font-ui text-meta text-mute">
                    <KindMark
                      className="size-3.5 text-accent"
                      kind={item.kind}
                    />
                    {KIND_LABELS[item.kind]}
                    <span aria-hidden="true">·</span>
                    <span suppressHydrationWarning>
                      {remainingDeletionWindow(item.recoverableUntil)}
                    </span>
                    <span aria-hidden="true">·</span>
                    Restorable until{" "}
                    <time dateTime={item.recoverableUntil}>
                      {restoreDeadline(item.recoverableUntil)}
                    </time>
                  </p>
                </div>
                <Button
                  disabled={pending !== null}
                  loading={pending === item.id}
                  onClick={() => restore(item)}
                  variant="outline"
                >
                  <RotateCcw aria-hidden="true" />
                  Restore
                </Button>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-6 rounded-plate bg-deep px-5 py-8 text-center font-ui text-ui text-mute">
            Nothing is waiting to be restored.
          </p>
        )}
      </div>
    </Shell>
  );
}
