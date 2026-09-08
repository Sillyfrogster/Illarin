"use client";

import { AlertCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { WORKING_COPY_STALE } from "@/lib/working-copy";
import styles from "./WorkingCopyNotice.module.css";

/** Says that this page's copy is behind, and leaves the catching up to its creator. */
export function WorkingCopyNotice() {
  const [stale, setStale] = useState(false);

  useEffect(() => {
    const behind = () => setStale(true);
    window.addEventListener(WORKING_COPY_STALE, behind);
    return () => window.removeEventListener(WORKING_COPY_STALE, behind);
  }, []);

  if (!stale) return null;

  return (
    <div className={styles.notice} role="alert">
      <AlertCircle size={17} aria-hidden="true" />
      <p>
        This asset was saved somewhere else, so nothing here can be saved until
        you catch up. What you have typed is still on the page. Copy anything
        worth keeping, then reload to work from the newer copy.
      </p>
      <button type="button" onClick={() => window.location.reload()}>
        Reload the page
      </button>
    </div>
  );
}
