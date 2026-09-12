"use client";

import { LockKeyhole, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { deleteAsset } from "@/lib/api/query";

export function DeleteControl({
  assetId,
  creator,
  kind,
  isDraft,
  frozen,
}: {
  assetId: string;
  creator: string;
  kind: string;
  isDraft: boolean;
  frozen: boolean;
}) {
  const router = useRouter();
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const noun = isDraft ? "draft" : "asset";

  async function remove() {
    if (pending || frozen) return;
    setPending(true);
    setMessage("");
    try {
      await deleteAsset(assetId);
      router.replace(`/@${creator}#deleted`);
      router.refresh();
    } catch {
      setMessage("The asset could not be deleted. Try again.");
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="delete-heading" className="flex gap-3">
      <span aria-hidden="true" className="mt-0.5 shrink-0 text-mute">
        {frozen ? <LockKeyhole size={18} /> : <Trash2 size={18} />}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div>
          <h3 className="text-ui font-medium text-ink" id="delete-heading">
            Delete this {noun}
          </h3>
          <p className="mt-1 text-meta text-mute">
            {frozen
              ? "A withheld asset cannot be deleted."
              : isDraft
                ? "Every block and image goes with it. You can restore it from your profile for 30 days."
                : "Its page and downloads stop now. You can restore it from your profile for 30 days."}
          </p>
          {isDraft && !frozen ? (
            <p className="mt-1 text-meta text-mute">
              A {kind} stays a {kind}. If you picked the wrong kind, delete this
              draft and start the one you meant.
            </p>
          ) : null}
        </div>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
        {confirming && !frozen ? (
          <div className="flex flex-col gap-3 rounded-plate bg-stop-wash p-4">
            <p className="text-ui text-ink">Move this {noun} to Deleted?</p>
            <div className="flex flex-wrap items-center gap-2">
              <Button loading={pending} onClick={remove} variant="stop">
                Delete asset
              </Button>
              <Button
                disabled={pending}
                onClick={() => setConfirming(false)}
                variant="ghost"
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Button
            className="self-start"
            disabled={frozen}
            onClick={() => setConfirming(true)}
          >
            {frozen ? "Deletion locked" : "Delete asset"}
          </Button>
        )}
      </div>
    </section>
  );
}
