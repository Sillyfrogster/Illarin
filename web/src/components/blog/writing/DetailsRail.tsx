"use client";

import { Field, TextInput } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import type {
  BlogCategory,
  Post,
  PostMedia,
  PostMediaPurpose,
} from "@/lib/api/query";
import { useBlogAddress } from "@/lib/origins";
import { normalizedSlug } from "@/lib/post-link";
import type { Draft } from "@/lib/post-writing";
import { PicturePicker } from "./PicturePicker";
import { PublishedIdentity } from "./PublishedIdentity";

export function DetailsRail({
  admin,
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
  categories: BlogCategory[];
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
  const blogAddress = useBlogAddress();

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
          hint={`${blogAddress}/${normalizedSlug(draft.slug) || "…"}. You can change this until you publish. After that it is fixed.`}
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

      <PicturePicker
        chosen={draft.header}
        describe
        hint="Optional image shown above the article."
        label="Header image"
        media={media}
        onChange={(header) => onChange({ header })}
        onUpload={onUpload}
        purpose="header"
      />

      <PicturePicker
        chosen={
          draft.linkCardMediaId
            ? { mediaId: draft.linkCardMediaId, alt: "", caption: "" }
            : null
        }
        describe={false}
        hint="Optional. Without one, Illarin composes a card from the title."
        label="Link card image"
        media={media}
        onChange={(card) =>
          onChange({ linkCardMediaId: card?.mediaId ?? null })
        }
        onUpload={onUpload}
        purpose="link_card"
      />

      {locked ? (
        <PublishedIdentity
          admin={admin}
          onCorrected={onCorrected}
          post={post}
        />
      ) : (
        <p className="font-prose text-meta text-mute">
          First publication records your name and picture in the byline.
        </p>
      )}
    </div>
  );
}
