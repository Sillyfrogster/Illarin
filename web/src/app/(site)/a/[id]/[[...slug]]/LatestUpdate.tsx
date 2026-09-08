import { ArrowRight } from "lucide-react";
import Link from "next/link";
import type { RecordedVersion } from "@/lib/api/query";
import { versionDate, versionSummary, versionTitle } from "@/lib/asset-updates";
import { assetHistoryHref } from "@/lib/asset-url";
import styles from "./LatestUpdate.module.css";

/** What readers have, and the way into everything Illarin recorded before it. */
export function LatestUpdate({
  assetId,
  kind,
  version,
}: {
  assetId: string;
  kind: string;
  version: RecordedVersion;
}) {
  return (
    <section className={styles.band} aria-labelledby="latest-update">
      <p className={styles.head}>
        <span className={styles.title} id="latest-update">
          {versionTitle(version)}
        </span>
        <time dateTime={version.recordedAt}>{versionDate(version)}</time>
        {version.versionLabel ? (
          <span>Creator's version {version.versionLabel}</span>
        ) : null}
      </p>
      <p className={styles.summary}>{versionSummary(version, kind)}</p>
      <Link href={assetHistoryHref(assetId)} className={styles.link}>
        Update history
        <ArrowRight size={15} aria-hidden="true" />
      </Link>
    </section>
  );
}
