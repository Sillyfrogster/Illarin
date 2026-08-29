"use client";

import { useEffect } from "react";
import { Field } from "@/components/console/Field";
import type {
  Post,
  PublicationApp,
  PublicationCategory,
} from "@/lib/api/query";
import styles from "./PostDetails.module.css";

type Release = { appId: string; version: string; address: string };

type Draft = {
  categoryId: string;
  slug: string;
  release: Release | null;
};

export function PostDetails({
  apps,
  categories,
  draft,
  locked,
  post,
  onChange,
}: {
  apps: PublicationApp[];
  categories: PublicationCategory[];
  draft: Draft;
  locked: boolean;
  post: Post;
  onChange: (patch: Partial<Draft>) => void;
}) {
  const category = categories.find((one) => one.id === draft.categoryId);
  const releasing = category?.slug === "release";

  useEffect(() => {
    if (!releasing || draft.release || apps.length === 0) return;
    onChange({ release: { appId: apps[0].id, version: "", address: "" } });
  }, [releasing, draft.release, apps, onChange]);

  return (
    <aside className={styles.details}>
      <h2>Details</h2>
      <Field htmlFor="post-category" label="Category">
        <select
          id="post-category"
          onChange={(event) => onChange({ categoryId: event.target.value })}
          value={draft.categoryId}
        >
          {categories.map((one) => (
            <option key={one.id} value={one.id}>
              {one.label}
            </option>
          ))}
        </select>
      </Field>
      <Field
        hint={
          locked
            ? "The address of a published post is fixed. An admin can correct it."
            : "The permanent address once the post is published."
        }
        htmlFor="post-slug"
        label="Address"
      >
        <span className={styles.address}>
          <span aria-hidden="true">/blog/</span>
          <input
            id="post-slug"
            maxLength={80}
            onChange={(event) => onChange({ slug: event.target.value })}
            value={draft.slug}
          />
        </span>
      </Field>
      {releasing ? (
        <ReleaseFields
          apps={apps}
          onChange={onChange}
          release={draft.release ?? { appId: "", version: "", address: "" }}
        />
      ) : null}
      {post.app ? (
        <p className={styles.attribution}>
          This post carries your name and {post.app.name}. Illarin stays the
          publisher.
        </p>
      ) : (
        <p className={styles.attribution}>
          This post carries the Illarin Team byline and your public positions.
        </p>
      )}
    </aside>
  );
}

function ReleaseFields({
  apps,
  release,
  onChange,
}: {
  apps: PublicationApp[];
  release: Release;
  onChange: (patch: Partial<Draft>) => void;
}) {
  return (
    <>
      <Field htmlFor="release-app" label="Project">
        <select
          id="release-app"
          onChange={(event) =>
            onChange({ release: { ...release, appId: event.target.value } })
          }
          value={release.appId}
        >
          {apps.map((app) => (
            <option key={app.id} value={app.id}>
              {app.name}
            </option>
          ))}
        </select>
      </Field>
      <Field htmlFor="release-version" label="Version">
        <input
          id="release-version"
          maxLength={40}
          onChange={(event) =>
            onChange({ release: { ...release, version: event.target.value } })
          }
          placeholder="2.4.0"
          value={release.version}
        />
      </Field>
      <Field
        hint="Optional. The canonical page for this release."
        htmlFor="release-address"
        label="Release notes"
      >
        <input
          id="release-address"
          onChange={(event) =>
            onChange({ release: { ...release, address: event.target.value } })
          }
          placeholder="https://"
          type="url"
          value={release.address}
        />
      </Field>
    </>
  );
}
