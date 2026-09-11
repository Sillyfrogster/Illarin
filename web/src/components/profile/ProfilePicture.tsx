"use client";

import { ImageUp, Trash2 } from "lucide-react";
import { useRef } from "react";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import { Button } from "@/components/ui/button";
import type { Profile } from "@/lib/api/query";

export function ProfilePicture({
  confirmingRemoval,
  onConfirmRemoval,
  onRemove,
  onUpload,
  pending,
  profile,
}: {
  confirmingRemoval: boolean;
  onConfirmRemoval: (confirming: boolean) => void;
  onRemove: () => void;
  onUpload: (file: File) => void;
  pending: boolean;
  profile: Profile;
}) {
  const picker = useRef<HTMLInputElement>(null);

  return (
    <div className="flex flex-wrap items-start gap-x-6 gap-y-5">
      <CreatorPortrait
        handle={profile.handle}
        picture={profile.avatar}
        size="lg"
      />
      <div className="min-w-0 flex-1 basis-64">
        <h2 className="font-ui text-ui font-medium text-ink">Picture</h2>
        <p className="mt-1 max-w-[44ch] font-prose text-ui text-mute">
          Upload a PNG, JPEG, WebP or GIF. Square images work best.
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Button
            disabled={pending}
            onClick={() => picker.current?.click()}
            variant="secondary"
          >
            <ImageUp aria-hidden="true" />
            {pending
              ? "Uploading…"
              : profile.avatar
                ? "Replace image"
                : "Upload image"}
          </Button>

          {profile.avatar && !confirmingRemoval ? (
            <Button
              disabled={pending}
              onClick={() => onConfirmRemoval(true)}
              variant="ghost"
            >
              <Trash2 aria-hidden="true" />
              Remove
            </Button>
          ) : null}

          {profile.avatar && confirmingRemoval ? (
            <span className="flex flex-wrap items-center gap-2">
              <span className="font-ui text-meta text-mute">
                Remove your profile picture? Your default avatar will appear
                instead.
              </span>
              <Button disabled={pending} onClick={onRemove} variant="stop">
                Remove
              </Button>
              <Button onClick={() => onConfirmRemoval(false)} variant="ghost">
                Keep
              </Button>
            </span>
          ) : null}
        </div>

        <input
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="sr-only"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) onUpload(file);
            event.target.value = "";
          }}
          ref={picker}
          type="file"
        />
      </div>
    </div>
  );
}
