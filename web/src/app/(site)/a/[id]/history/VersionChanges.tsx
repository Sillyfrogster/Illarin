"use client";

import Image from "next/image";
import { useEffect, useState } from "react";
import {
  compareAssetVersions,
  type VersionChange,
  type VersionComparison,
} from "@/lib/api/query";
import { wordDiff } from "@/lib/text-diff";
import styles from "./HistoryPage.module.css";

const MARKS: Record<VersionChange["kind"], string> = {
  addition: "Added",
  removal: "Removed",
  change: "Changed",
};

/** What Illarin computed between two recorded versions, apart from anything the creator wrote. */
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

  if (reading) return <p className={styles.quiet}>Reading the comparison…</p>;
  if (!compared) return <p className={styles.refusal}>{refusal}</p>;
  if (compared.unavailable) {
    return <p className={styles.refusal}>{compared.unavailable}</p>;
  }

  return (
    <div className={styles.comparison}>
      {compared.promptsWithheld ? (
        <p className={styles.withheld}>
          This {kind} keeps its prompts for linked applications, so their
          wording is not shown here.
        </p>
      ) : null}

      {compared.groups.length === 0 ? (
        <p className={styles.quiet}>
          Nothing differs between these two versions.
        </p>
      ) : (
        compared.groups.map((group) => (
          <section key={group.subject} className={styles.group}>
            <h3>{group.label}</h3>
            <ul>
              {group.changes.map((change, index) => (
                <li
                  key={`${group.subject}-${index}`}
                  className={styles.changeRow}
                >
                  <ChangeRow change={change} subject={group.label} />
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
    </div>
  );
}

/** One addition, removal or edit, with its detail behind a disclosure. */
function ChangeRow({
  change,
  subject,
}: {
  change: VersionChange;
  subject: string;
}) {
  const hasText = Boolean(change.before || change.after);
  const hasImage = Boolean(change.beforeImage || change.afterImage);

  return (
    <>
      <p className={styles.changeHead}>
        <span className={styles[change.kind]}>{MARKS[change.kind]}</span>
        <span className={styles.changeName}>{change.name || subject}</span>
        {change.previousName ? (
          <span className={styles.renamed}>was {change.previousName}</span>
        ) : null}
      </p>

      {hasText || hasImage ? (
        <details className={styles.detail}>
          <summary>
            {hasImage ? "Show the pictures" : "Show the wording"}
          </summary>
          {hasImage ? (
            <div className={styles.pictures}>
              {change.beforeImage ? (
                <figure>
                  <Image
                    src={change.beforeImage}
                    alt=""
                    width={160}
                    height={160}
                    unoptimized
                  />
                  <figcaption>Before</figcaption>
                </figure>
              ) : null}
              {change.afterImage ? (
                <figure>
                  <Image
                    src={change.afterImage}
                    alt=""
                    width={160}
                    height={160}
                    unoptimized
                  />
                  <figcaption>After</figcaption>
                </figure>
              ) : null}
            </div>
          ) : (
            <Wording before={change.before ?? ""} after={change.after ?? ""} />
          )}
        </details>
      ) : null}
    </>
  );
}

/** One text change, with the words an update took out and the words it put in. */
function Wording({ before, after }: { before: string; after: string }) {
  return (
    <p className={styles.wording}>
      {wordDiff(before, after).map((piece, index) => {
        // The pieces have no identity of their own, so their place in the text is the key.
        const key = `${index}-${piece.kind}`;
        if (piece.kind === "removed") return <del key={key}>{piece.text}</del>;
        if (piece.kind === "added") return <ins key={key}>{piece.text}</ins>;
        return <span key={key}>{piece.text}</span>;
      })}
    </p>
  );
}
