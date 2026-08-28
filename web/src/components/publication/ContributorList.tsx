"use client";

import { Plus, UserRound } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import type {
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
} from "@/lib/api/query";
import { ContributorDialog } from "./ContributorDialog";
import styles from "./ContributorList.module.css";
import hub from "./PublicationHub.module.css";

export function ContributorList({
  grants,
  apps,
  categories,
  onChanged,
  onReload,
  onFailure,
}: {
  grants: PublicationGrant[];
  apps: PublicationApp[];
  categories: PublicationCategory[];
  onChanged: (grants: PublicationGrant[]) => void;
  onReload: () => void;
  onFailure: (message: string) => void;
}) {
  const [editing, setEditing] = useState<PublicationGrant | null>(null);
  const [approving, setApproving] = useState(false);

  const active = grants.filter((grant) => grant.active);
  const ended = grants.filter((grant) => !grant.active);
  const open = apps.filter((app) => !app.retired);

  return (
    <Section
      title="Contributors"
      count={active.length}
      wide
      action={
        <button
          type="button"
          className={hub.add}
          onClick={() => setApproving(true)}
          disabled={open.length === 0}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          Approve someone
        </button>
      }
      retiredLabel={ended.length > 0 ? `${ended.length} ended` : undefined}
      retired={
        ended.length > 0 ? (
          <ul className={rows.pastList}>
            {ended.map((grant) => (
              <li className={rows.pastRow} key={grant.id}>
                <span>
                  @{grant.holder.handle} for {grant.app.name}
                </span>
                {grant.revokedAt ? shownDate(grant.revokedAt) : null}
              </li>
            ))}
          </ul>
        ) : null
      }
    >
      {active.length === 0 ? (
        <p className={rows.empty}>
          {open.length === 0
            ? "Add an app first. An approval binds one person to one app."
            : "Nobody is approved to publish. Illarin's own writing still works."}
        </p>
      ) : (
        <ul className={styles.people}>
          {active.map((grant) => (
            <li className={styles.person} key={grant.id}>
              <span className={styles.portrait}>
                {grant.holder.avatar ? (
                  <Image
                    src={grant.holder.avatar.url}
                    alt=""
                    width={44}
                    height={44}
                    unoptimized
                  />
                ) : (
                  <UserRound size={19} strokeWidth={1.5} aria-hidden="true" />
                )}
              </span>
              <span className={styles.who}>
                {grant.holder.displayName ? (
                  <span className={styles.said}>
                    {grant.holder.displayName}
                  </span>
                ) : null}
                <Link
                  className={styles.handle}
                  href={`/${grant.holder.handle}`}
                >
                  @{grant.holder.handle}
                </Link>
              </span>
              <span className={styles.allowed}>
                <strong>{grant.app.name}</strong>
                {grant.app.retired ? " (retired)" : null} ·{" "}
                {grant.categories.map((category) => category.label).join(", ")}{" "}
                · {grant.defaultCategory.label} by default
              </span>
              <span className={rows.actions}>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setEditing(grant)}
                >
                  Edit
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      {approving ? (
        <ContributorDialog
          key="approving"
          existing={null}
          apps={open}
          categories={categories}
          onClose={() => setApproving(false)}
          onSaved={(saved) => onChanged([saved, ...grants])}
          onRevoked={onReload}
          onFailure={onFailure}
        />
      ) : null}
      {editing ? (
        <ContributorDialog
          key={editing.id}
          existing={editing}
          apps={open}
          categories={categories}
          onClose={() => setEditing(null)}
          onSaved={(saved) =>
            onChanged(grants.map((one) => (one.id === saved.id ? saved : one)))
          }
          onRevoked={onReload}
          onFailure={onFailure}
        />
      ) : null}
    </Section>
  );
}

function shownDate(when: string) {
  return new Date(when).toLocaleDateString(undefined, {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
