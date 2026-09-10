"use client";

import { Eye, EyeOff, LockKeyhole } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { AssetDetail } from "@/lib/api/query";
import { saveAssetDiscovery } from "@/lib/api/query";

/** Whether a published asset appears in the catalog. */
export function DiscoveryControl({
  assetId,
  initialDiscovery,
  frozen,
}: {
  assetId: string;
  initialDiscovery: AssetDetail["discovery"];
  frozen: boolean;
}) {
  const router = useRouter();
  const [discovery, setDiscovery] = useState(initialDiscovery);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  const listed = discovery === "listed";
  const next = listed ? "unlisted" : "listed";

  async function changeDiscovery() {
    setPending(true);
    setMessage("");
    setDiscovery(next);
    try {
      await saveAssetDiscovery(assetId, next);
      router.refresh();
    } catch {
      setDiscovery(discovery);
      setMessage("Discovery could not be changed. Try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="discovery-heading" className="flex gap-3">
      <span aria-hidden="true" className="mt-0.5 shrink-0 text-mute">
        {frozen ? (
          <LockKeyhole size={18} />
        ) : listed ? (
          <Eye size={18} />
        ) : (
          <EyeOff size={18} />
        )}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div>
          <h3 className="text-ui font-medium text-ink" id="discovery-heading">
            Catalog discovery
          </h3>
          <p className="mt-1 text-meta text-mute">
            {frozen
              ? "Locked while this asset is withheld. Only an admin can remove the withhold."
              : listed
                ? "Listed in the catalog and on your public profile."
                : "Unlisted from discovery. Anyone with the link can still view and download it."}
          </p>
        </div>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
        <Button
          className="self-start"
          disabled={frozen}
          loading={pending}
          onClick={changeDiscovery}
        >
          {frozen
            ? "Discovery locked"
            : listed
              ? "Make unlisted"
              : "List in catalog"}
        </Button>
      </div>
    </section>
  );
}
