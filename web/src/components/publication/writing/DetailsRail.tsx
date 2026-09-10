"use client";

import { useEffect } from "react";
import { Field, TextInput } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import type {
  Post,
  PostMedia,
  PostMediaPurpose,
  PublicationApp,
  PublicationCategory,
} from "@/lib/api/query";
import { normalizedSlug } from "@/lib/post-link";
import type { Draft } from "@/lib/post-writing";
import { PicturePicker } from "./PicturePicker";
import { PublishedIdentity } from "./PublishedIdentity";

type Release = NonNullable<Draft["release"]>;

export function DetailsRail({
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
    <div className="flex flex-col gap-7">
      <Field htmlFor="post-category" label="Category">
        <Select
          className="w-full"
          id="post-category"
          onChange={(event) => onChange({ categoryId: event.target.value })}
          value={draft.categoryId}
        >
          {categories.map((one) => (
            <option key={one.id} value={one.id}>
              {one.label}
            </option>
          ))}
        </Select>
      </Field>

      {locked ? null : (
        <Field
          hint={`illarin.xyz/blog/${normalizedSlug(draft.slug) || "…"}. You can change this until you publish. After that it is fixed.`}
          htmlFor="post-slug"
          label="Address"
        >
          <TextInput
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
      ) : (
        <p className="font-prose text-meta text-mute">
          {post.app
            ? `When you publish, this post takes your name and ${post.app.name}. Illarin stays the publisher.`
            : "When you publish, this post takes your name, your jobs at Illarin and the Illarin Team line."}
        </p>
      )}
    </div>
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
        <Select
          className="w-full"
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
        </Select>
      </Field>
      <Field className="max-w-40" htmlFor="release-version" label="Version">
        <TextInput
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
        <TextInput
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
