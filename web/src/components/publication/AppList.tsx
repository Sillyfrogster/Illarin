"use client";

import { ArrowDown, ArrowUp, Package, Plus } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import { orderApps, updateApp } from "@/lib/api/publication";
import type { PublicationApp } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import { AppDialog } from "./AppDialog";
import styles from "./PublicationHub.module.css";

export function AppList({
  apps,
  onChanged,
  onFailure,
}: {
  apps: PublicationApp[];
  onChanged: (apps: PublicationApp[]) => void;
  onFailure: (message: string) => void;
}) {
  const [editing, setEditing] = useState<PublicationApp | null>(null);
  const [adding, setAdding] = useState(false);

  const current = apps.filter((app) => !app.retired);
  const retired = apps.filter((app) => app.retired);

  function replace(saved: PublicationApp, added: boolean) {
    onChanged(
      added
        ? [...apps, saved]
        : apps.map((app) => (app.id === saved.id ? saved : app)),
    );
    if (editing?.id === saved.id) setEditing(saved);
  }

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (target < 0 || target >= current.length) return;
    const answer = await orderApps(
      moved(
        apps.map((app) => app.id),
        apps.indexOf(current[index]),
        apps.indexOf(current[target]),
      ),
    );
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(answer.value.apps);
  }

  async function bringBack(app: PublicationApp) {
    const answer = await updateApp(app.id, { retired: false });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    replace(answer.value, false);
  }

  return (
    <Section
      title="Apps"
      count={current.length}
      action={
        <button
          type="button"
          className={styles.add}
          onClick={() => setAdding(true)}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          Add an app
        </button>
      }
      retiredLabel={
        retired.length > 0 ? `${retired.length} retired` : undefined
      }
      retired={
        retired.length > 0 ? (
          <ul className={rows.pastList}>
            {retired.map((app) => (
              <li className={rows.pastRow} key={app.id}>
                <span>{app.name}</span>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => bringBack(app)}
                >
                  Bring back
                </button>
              </li>
            ))}
          </ul>
        ) : null
      }
    >
      {current.length === 0 ? (
        <p className={rows.empty}>
          No app is configured, so nobody can be approved yet.
        </p>
      ) : (
        <ol className={rows.list}>
          {current.map((app, index) => (
            <li className={rows.row} key={app.id}>
              <span className={rows.mark} data-blank={!app.mark || undefined}>
                {app.mark ? (
                  <Image
                    src={app.mark.url}
                    alt=""
                    width={34}
                    height={34}
                    unoptimized
                  />
                ) : (
                  <Package size={17} strokeWidth={1.6} aria-hidden="true" />
                )}
              </span>
              <span className={rows.name}>
                {app.name} <span className={rows.slug}>{app.slug}</span>
              </span>
              <span className={rows.detail}>
                <a href={app.home} rel="noreferrer noopener" target="_blank">
                  {app.home.replace(/^https:\/\//, "")}
                </a>
              </span>
              <span className={rows.actions}>
                <button
                  type="button"
                  className={rows.iconButton}
                  onClick={() => reorder(index, -1)}
                  disabled={index === 0}
                  aria-label={`Move ${app.name} up`}
                >
                  <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={rows.iconButton}
                  onClick={() => reorder(index, 1)}
                  disabled={index === current.length - 1}
                  aria-label={`Move ${app.name} down`}
                >
                  <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setEditing(app)}
                >
                  Edit
                </button>
              </span>
            </li>
          ))}
        </ol>
      )}

      {adding ? (
        <AppDialog
          key="adding"
          existing={null}
          onClose={() => setAdding(false)}
          onSaved={replace}
          onFailure={onFailure}
        />
      ) : null}
      {editing ? (
        <AppDialog
          key={editing.id}
          existing={editing}
          onClose={() => setEditing(null)}
          onSaved={replace}
          onFailure={onFailure}
        />
      ) : null}
    </Section>
  );
}
