"use client";

import { useState } from "react";
import type { RecordedVersion } from "@/lib/api/query";
import { versionDate, versionSummary, versionTitle } from "@/lib/asset-updates";
import styles from "./HistoryPage.module.css";
import { VersionChanges } from "./VersionChanges";

/** The versions an asset has recorded, and the comparison between any two of them. */
export function UpdateHistory({
  assetId,
  kind,
  versions,
}: {
  assetId: string;
  kind: string;
  versions: RecordedVersion[];
}) {
  const newest = versions[0];
  const [from, setFrom] = useState(versions[1]?.number ?? 0);
  const [to, setTo] = useState(newest?.number ?? 0);

  if (versions.length === 0) {
    return (
      <p className={styles.quiet}>
        Illarin has recorded no versions of this {kind} yet. History begins at
        its first publication.
      </p>
    );
  }

  return (
    <>
      {versions.length > 1 ? (
        <section className={styles.compare} aria-labelledby="compare-heading">
          <h2 id="compare-heading">Compare two versions</h2>
          <div className={styles.pair}>
            <VersionPicker
              words="Earlier"
              versions={versions}
              chosen={from}
              onChoose={setFrom}
            />
            <VersionPicker
              words="Later"
              versions={versions}
              chosen={to}
              onChoose={setTo}
            />
          </div>
          <VersionChanges
            assetId={assetId}
            kind={kind}
            from={Math.min(from, to)}
            to={Math.max(from, to)}
          />
        </section>
      ) : null}

      <ol className={styles.versions}>
        {versions.map((version, index) => {
          const previous = versions[index + 1];
          return (
            <li key={version.id} className={styles.version}>
              <div className={styles.versionHead}>
                <h2>{versionTitle(version)}</h2>
                <p className={styles.recorded}>
                  <time dateTime={version.recordedAt}>
                    {versionDate(version)}
                  </time>
                  {version.versionLabel ? (
                    <span className={styles.label}>
                      Creator's version {version.versionLabel}
                    </span>
                  ) : null}
                </p>
              </div>

              <p className={styles.summary}>{versionSummary(version, kind)}</p>
              {version.notes ? (
                <p className={styles.notes}>{version.notes}</p>
              ) : null}

              {previous ? (
                <ChangesDisclosure
                  assetId={assetId}
                  kind={kind}
                  from={previous.number}
                  to={version.number}
                  words={`What changed since ${versionTitle(previous).toLowerCase()}`}
                />
              ) : (
                <p className={styles.quiet}>
                  Illarin recorded nothing before this, so there is nothing to
                  compare it with.
                </p>
              )}
            </li>
          );
        })}
      </ol>
    </>
  );
}

/** One side of the pair a reader is comparing. */
function VersionPicker({
  words,
  versions,
  chosen,
  onChoose,
}: {
  words: string;
  versions: RecordedVersion[];
  chosen: number;
  onChoose: (number: number) => void;
}) {
  return (
    <label>
      <span>{words}</span>
      <select
        value={chosen}
        onChange={(event) => onChoose(Number(event.target.value))}
      >
        {versions.map((version) => (
          <option key={version.id} value={version.number}>
            {versionTitle(version)} · {versionDate(version)}
          </option>
        ))}
      </select>
    </label>
  );
}

/** One version's changes, read only once a reader opens them. */
function ChangesDisclosure({
  assetId,
  kind,
  from,
  to,
  words,
}: {
  assetId: string;
  kind: string;
  from: number;
  to: number;
  words: string;
}) {
  const [opened, setOpened] = useState(false);

  return (
    <details
      className={styles.changes}
      onToggle={(event) => {
        if (event.currentTarget.open) setOpened(true);
      }}
    >
      <summary>{words}</summary>
      {opened ? (
        <VersionChanges assetId={assetId} kind={kind} from={from} to={to} />
      ) : null}
    </details>
  );
}
