"use client";

import { FileUp } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import {
  fetchAsset,
  fetchWaitingReplacement,
  type IngestOperation,
} from "@/lib/api/query";
import { WORKING_COPY_SAVED } from "@/lib/working-copy";
import { PublishUpdateDialog } from "./PublishUpdateDialog";
import { ReplaceFileDialog } from "./ReplaceFileDialog";
import styles from "./UpdatePanel.module.css";

/** Where the creator of a published asset prepares and publishes its next update. */
export function UpdatePanel({
  assetId,
  kind,
  unpublishedChanges,
}: {
  assetId: string;
  kind: string;
  unpublishedChanges: boolean;
}) {
  const router = useRouter();
  const [changed, setChanged] = useState(unpublishedChanges);
  const [waiting, setWaiting] = useState<IngestOperation | null>(null);
  const [open, setOpen] = useState<"publish" | "replace" | null>(null);

  useEffect(() => setChanged(unpublishedChanges), [unpublishedChanges]);

  const readStanding = useCallback(async () => {
    const page = await fetchAsset(assetId, undefined, true);
    if (page) setChanged(Boolean(page.unpublishedChanges));
  }, [assetId]);

  useEffect(() => {
    const saved = () => void readStanding();
    window.addEventListener(WORKING_COPY_SAVED, saved);
    return () => window.removeEventListener(WORKING_COPY_SAVED, saved);
  }, [readStanding]);

  useEffect(() => {
    let reading = true;
    void fetchWaitingReplacement(assetId).then((found) => {
      if (reading) setWaiting(found);
    });
    return () => {
      reading = false;
    };
  }, [assetId]);

  function settled(operation: IngestOperation | null) {
    setWaiting(operation);
    setOpen(null);
    void readStanding();
    router.refresh();
  }

  return (
    <section className={styles.panel} aria-labelledby="update-heading">
      <div>
        <h2 id="update-heading">Your working copy</h2>
        <p>{standing(waiting, changed)}</p>
      </div>

      <div className={styles.actions}>
        <button
          type="button"
          className={styles.publish}
          onClick={() => setOpen("publish")}
        >
          Publish update
        </button>
        <button
          type="button"
          className={styles.replace}
          onClick={() => setOpen("replace")}
        >
          <FileUp size={15} aria-hidden="true" />
          {waiting ? "Review the uploaded file" : "Replace the file"}
        </button>
      </div>

      {open === "publish" ? (
        <PublishUpdateDialog
          assetId={assetId}
          kind={kind}
          changed={changed}
          replacementWaiting={Boolean(waiting)}
          onClose={() => setOpen(null)}
          onPublished={() => settled(null)}
        />
      ) : null}

      {open === "replace" ? (
        <ReplaceFileDialog
          assetId={assetId}
          waiting={waiting}
          onClose={() => setOpen(null)}
          onWaiting={setWaiting}
          onAccepted={() => settled(null)}
          onDiscarded={() => settled(null)}
        />
      ) : null}
    </section>
  );
}

/** What the page tells a creator about the version readers have. */
function standing(waiting: IngestOperation | null, changed: boolean): string {
  if (waiting?.status === "preview") {
    return "An uploaded file is waiting for your review. Nothing readers see changes until you accept or discard it.";
  }
  if (waiting) {
    return "Illarin is reading the file you uploaded. Readers keep the published version while it works.";
  }
  if (changed) {
    return "You have changes readers cannot see. They stay private until you publish an update.";
  }
  return "Readers see everything on this page. Anything you change here stays private until you publish an update.";
}
