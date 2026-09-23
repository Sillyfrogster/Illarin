"use client";

import { ImageUp, Trash2 } from "lucide-react";
import Image from "next/image";
import { type DragEvent, useRef, useState } from "react";
import { CropPicture, cropsCleanly } from "@/components/media/CropPicture";
import { Button } from "@/components/ui/button";
import type { Profile } from "@/lib/api/query";
import { cn } from "@/lib/cn";

/** Wide enough for a desktop banner, tall enough that a phone still sees the middle of it. */
const BANNER_ASPECT = 3;

const ACCEPTS = "image/png,image/jpeg,image/webp,image/gif";

/** The banner across the top of a profile, or the creator's tint until they add one */
export function Banner({
  banner,
  editing,
  onPick,
  onRemove,
  pending,
}: {
  banner: Profile["banner"];
  editing: boolean;
  onPick: (file: File) => void;
  onRemove: () => void;
  pending: boolean;
}) {
  const picker = useRef<HTMLInputElement>(null);
  const [framing, setFraming] = useState<File | null>(null);
  const [dragging, setDragging] = useState(false);

  function choose(file: File | undefined) {
    if (!file) return;
    if (cropsCleanly(file)) setFraming(file);
    else onPick(file);
  }

  function drop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setDragging(false);
    choose(event.dataTransfer.files[0]);
  }

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: dropping is a shortcut for the banner buttons, which do the same from the keyboard
    <div
      className={cn(
        "relative h-[clamp(9.5rem,22vw,14rem)] w-full overflow-hidden",
        dragging && "outline-2 -outline-offset-2 outline-accent",
      )}
      onDragLeave={editing ? () => setDragging(false) : undefined}
      onDragOver={
        editing
          ? (event) => {
              event.preventDefault();
              setDragging(true);
            }
          : undefined
      }
      onDrop={editing ? drop : undefined}
    >
      {banner ? (
        <Image
          alt=""
          className="size-full object-cover"
          fill
          priority
          sizes="100vw"
          src={banner.url}
          unoptimized
        />
      ) : (
        <div
          aria-hidden="true"
          className="size-full [background:radial-gradient(120%_140%_at_18%_110%,color-mix(in_oklab,var(--tint)_55%,transparent),transparent_60%),radial-gradient(80%_120%_at_88%_-10%,color-mix(in_oklab,var(--tint)_35%,transparent),transparent_60%),color-mix(in_oklab,var(--v-field)_82%,var(--tint))]"
        />
      )}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 bottom-0 h-1/2 bg-linear-to-t from-[color-mix(in_oklab,var(--v-field)_35%,transparent)] to-transparent"
      />

      {editing ? (
        <div className="absolute top-8 right-[var(--rail-inset)] flex items-center gap-2 [--v-deep:rgb(255_255_255/0.14)] [--v-ink:#ffffff] [--v-mute:#e6e0f0] lg:top-auto lg:right-auto lg:bottom-4 lg:left-[var(--rail-inset)]">
          <Button
            className="bg-[rgb(0_0_0/0.45)] backdrop-blur-md hover:bg-[rgb(0_0_0/0.6)] max-lg:size-11 max-lg:px-0"
            disabled={pending}
            id="profile-banner-pick"
            loading={pending}
            onClick={() => picker.current?.click()}
            variant="secondary"
          >
            <ImageUp aria-hidden="true" />
            <span className="max-lg:sr-only">
              {banner ? "Change banner" : "Add banner"}
            </span>
          </Button>
          {banner ? (
            <Button
              className="bg-[rgb(0_0_0/0.45)] backdrop-blur-md hover:bg-[rgb(0_0_0/0.6)] max-lg:size-11 max-lg:px-0"
              disabled={pending}
              onClick={onRemove}
              variant="secondary"
            >
              <Trash2 aria-hidden="true" />
              <span className="max-lg:sr-only">Remove banner</span>
            </Button>
          ) : null}
          <input
            accept={ACCEPTS}
            className="sr-only"
            onChange={(event) => {
              choose(event.target.files?.[0]);
              event.target.value = "";
            }}
            ref={picker}
            type="file"
          />
        </div>
      ) : null}

      <CropPicture
        aspect={BANNER_ASPECT}
        file={framing}
        onCancel={() => setFraming(null)}
        onDone={(file) => {
          setFraming(null);
          onPick(file);
        }}
        title="Frame your banner"
      />
    </div>
  );
}
