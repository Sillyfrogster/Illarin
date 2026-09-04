"use client";

import { useEffect } from "react";
import { Field } from "@/components/console/Field";
import type {
  Post,
  PostMedia,
  PostMediaPurpose,
  PublicationApp,
  PublicationCategory,
} from "@/lib/api/query";
import { type Chosen, PicturePicker } from "./PicturePicker";
import styles from "./PostDetails.module.css";
import { PublishedIdentity } from "./PublishedIdentity";

type Release = { appId: string; version: string; address: string };

type Draft = {
  categoryId: string;
  slug: string;
  release: Release | null;
  header: Chosen | null;
  socialMediaId: string | null;
};

export function PostDetails({
  admin,
  apps,
  categories,
  draft,
  locked,
  media,
  post,
  onChange,
  onCorrected,
  onUpload,
}: {
  admin: boolean;
  apps: PublicationApp[];
  categories: PublicationCategory[];
  draft: Draft;
  locked: boolean;
  media: PostMedia[];
  post: Post;
  onChange: (patch: Partial<Draft>) => void;
  onCorrected: (post: Post) => void;
  onUpload: (
    purpose: PostMediaPurpose,
    file: File,
  ) => Promise<PostMedia | null>;
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
      {locked ? null : (
        <Field
          hint={`illarin.xyz/blog/${draft.slug || "…"}. You can change this until you publish. After that it is fixed.`}
          htmlFor="post-slug"
          label="Address"
        >
          <input
            id="post-slug"
            maxLength={80}
            onChange={(event) => onChange({ slug: event.target.value })}
            value={draft.slug}
          />
        </Field>
      )}
      {releasing ? (
        <ReleaseFields
          apps={apps}
          onChange={onChange}
          release={draft.release ?? { appId: "", version: "", address: "" }}
        />
      ) : null}
      <PicturePicker
        chosen={draft.header}
        describe
        hint="Optional. Opens the article, above the body."
        label="Header picture"
        media={media}
        onChange={(header) => onChange({ header })}
        onUpload={onUpload}
        purpose="header"
      />
      <PicturePicker
        chosen={
          draft.socialMediaId
            ? { mediaId: draft.socialMediaId, alt: "", caption: "" }
            : null
        }
        describe={false}
        hint="Optional. Without one, Illarin composes a card from the title."
        label="Social image"
        media={media}
        onChange={(social) =>
          onChange({ socialMediaId: social?.mediaId ?? null })
        }
        onUpload={onUpload}
        purpose="social"
      />
      {locked ? (
        <PublishedIdentity
          admin={admin}
          onCorrected={onCorrected}
          post={post}
        />
      ) : post.app ? (
        <p className={styles.attribution}>
          When you publish, this post takes your name and {post.app.name}.
          Illarin stays the publisher.
        </p>
      ) : (
        <p className={styles.attribution}>
          When you publish, this post takes your name, your jobs at Illarin and
          the Illarin Team line.
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
