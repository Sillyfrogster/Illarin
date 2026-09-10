"use client";

import { ImageUp, Trash2 } from "lucide-react";
import Image from "next/image";
import { useId, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/field";
import type { PostMedia, PostMediaPurpose } from "@/lib/api/query";
import { POST_PICTURE_TEXT_LIMIT } from "@/lib/post-document";

export type Chosen = { mediaId: string; alt: string; caption: string };

/** One uploaded picture a post carries beside its body. */
export function PicturePicker({
  chosen,
  describe,
  hint,
  label,
  media,
  onChange,
  onUpload,
  purpose,
}: {
  chosen: Chosen | null;
  describe: boolean;
  hint: string;
  label: string;
  media: PostMedia[];
  onChange: (chosen: Chosen | null) => void;
  onUpload: (
    purpose: PostMediaPurpose,
    file: File,
  ) => Promise<PostMedia | null>;
  purpose: PostMediaPurpose;
}) {
  const field = useId();
  const chooser = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [refusal, setRefusal] = useState("");
  const picture = chosen && media.find((one) => one.id === chosen.mediaId);

  async function choose(file: File | undefined) {
    if (!file) return;
    setBusy(true);
    setRefusal("");
    const added = await onUpload(purpose, file);
    setBusy(false);
    if (!added) {
      setRefusal("That picture could not be uploaded. Try another file.");
      return;
    }
    onChange({
      mediaId: added.id,
      alt: chosen?.alt ?? "",
      caption: chosen?.caption ?? "",
    });
  }

  return (
    <section className="flex flex-col gap-3">
      <h3 className="font-display text-ui font-medium text-ink">{label}</h3>
      {picture ? (
        <Image
          alt=""
          className="h-auto w-full rounded-control bg-deep"
          height={picture.height}
          src={picture.thumbUrl}
          unoptimized
          width={picture.width}
        />
      ) : (
        <p className="font-prose text-meta text-mute">{hint}</p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          disabled={busy}
          onClick={() => chooser.current?.click()}
          size="compact"
        >
          <ImageUp aria-hidden="true" />
          {picture ? "Replace" : "Upload"}
        </Button>
        {picture ? (
          <Button onClick={() => onChange(null)} size="compact" variant="ghost">
            <Trash2 aria-hidden="true" />
            Remove
          </Button>
        ) : null}
      </div>
      <input
        accept="image/png,image/jpeg,image/webp,image/gif"
        aria-label={`${label} file`}
        className="sr-only"
        id={`${field}-file`}
        onChange={(event) => void choose(event.target.files?.[0])}
        ref={chooser}
        type="file"
      />
      {busy || refusal ? (
        <output
          className={
            refusal
              ? "font-prose text-meta text-stop"
              : "font-prose text-meta text-mute"
          }
        >
          {busy ? "Uploading…" : refusal}
        </output>
      ) : null}
      {picture && describe && chosen ? (
        <>
          <Field
            hint="Readers who cannot see the picture read this instead."
            htmlFor={`${field}-alt`}
            label="Description"
          >
            <TextInput
              aria-invalid={chosen.alt.trim() ? undefined : true}
              id={`${field}-alt`}
              maxLength={POST_PICTURE_TEXT_LIMIT}
              onChange={(event) =>
                onChange({ ...chosen, alt: event.target.value })
              }
              value={chosen.alt}
            />
          </Field>
          <Field
            hint="Optional. Shown under the picture."
            htmlFor={`${field}-caption`}
            label="Caption"
          >
            <TextInput
              id={`${field}-caption`}
              maxLength={POST_PICTURE_TEXT_LIMIT}
              onChange={(event) =>
                onChange({ ...chosen, caption: event.target.value })
              }
              value={chosen.caption}
            />
          </Field>
        </>
      ) : null}
    </section>
  );
}
