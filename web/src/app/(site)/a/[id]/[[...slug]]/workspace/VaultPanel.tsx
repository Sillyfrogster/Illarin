"use client";

import { Upload } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import {
  addWorkImage,
  discardVaultPicture,
  placeVaultPicture,
  type VaultPicture,
} from "@/lib/api/query";
import { useWorkingCopy } from "@/lib/working-copy";
import { Note, RemoveAction } from "./fields";
import { useWorkspace } from "./state";

const PLACE =
  "inline-flex min-h-11 items-center gap-2 rounded-control bg-deep px-4 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45";

export function VaultPanel({
  pictures,
  onRelease,
  onReload,
}: {
  pictures: VaultPicture[];
  onRelease: (pictureId: string) => void;
  onReload: () => void;
}) {
  const workspace = useWorkspace();
  const candidate = useWorkingCopy();
  const router = useRouter();
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const titles = new Map(
    workspace.blocks.map((block) => [block.id, block.title]),
  );

  async function run(pictureId: string, work: () => Promise<void>) {
    if (busy) return;
    setBusy(pictureId);
    setMessage("");
    try {
      await work();
      onRelease(pictureId);
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "That did not work. Try again.",
      );
    } finally {
      setBusy(null);
    }
  }

  function place(picture: VaultPicture, file?: File) {
    run(picture.id, async () => {
      let mediaId: string | undefined;
      if (file) {
        mediaId = await addWorkImage(
          candidate,
          workspace.workId,
          file,
          "gallery",
        );
      }
      const saved = await placeVaultPicture(
        candidate,
        workspace.workId,
        picture.id,
        mediaId,
      );
      workspace.applyServerBlocks(saved);
      workspace.say(`${picture.name || "Picture"} placed.`);
      if (!picture.blockId) onReload();
      if (file) router.refresh();
    });
  }

  function discard(picture: VaultPicture) {
    run(picture.id, () =>
      discardVaultPicture(candidate, workspace.workId, picture.id),
    );
  }

  if (pictures.length === 0) {
    return (
      <Note>Nothing is waiting. Every picture has been placed or let go.</Note>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <ol className="m-0 flex list-none flex-col gap-8 p-0">
        {pictures.map((picture) => (
          <WaitingPicture
            busy={busy === picture.id}
            key={picture.id}
            onDiscard={() => discard(picture)}
            onPlace={(file) => place(picture, file)}
            picture={picture}
            target={picture.blockId ? titles.get(picture.blockId) : undefined}
          />
        ))}
      </ol>
      {message ? (
        <output className="text-meta text-stop">{message}</output>
      ) : null}
    </div>
  );
}

function WaitingPicture({
  busy,
  onDiscard,
  onPlace,
  picture,
  target,
}: {
  busy: boolean;
  onDiscard: () => void;
  onPlace: (file?: File) => void;
  picture: VaultPicture;
  target?: string;
}) {
  const file = useRef<HTMLInputElement>(null);
  const name = picture.name.trim() || "Untitled picture";
  const destination = target
    ? `Goes into “${target}”`
    : `Starts a block called “${picture.section.trim() || "Pictures"}”`;

  return (
    <li className="flex flex-col gap-3">
      <p className="font-display text-ui font-medium text-ink">{name}</p>
      {picture.media ? (
        <div className="w-60 max-w-full overflow-hidden rounded-plate bg-media">
          <Image
            alt=""
            className="h-auto w-full"
            height={picture.media.height}
            sizes="240px"
            src={picture.media.thumbUrl}
            unoptimized
            width={picture.media.width}
          />
        </div>
      ) : (
        <p className="text-meta text-mute [overflow-wrap:anywhere]">
          Shown from {picture.address}. Illarin does not fetch it, so upload
          your own copy to keep it.
        </p>
      )}
      <p className="text-meta text-mute">{destination}</p>
      <div className="flex flex-wrap items-center gap-1">
        {picture.media ? (
          <button
            className={PLACE}
            disabled={busy}
            onClick={() => onPlace()}
            type="button"
          >
            Place it
          </button>
        ) : (
          <>
            <button
              className={PLACE}
              disabled={busy}
              onClick={() => file.current?.click()}
              type="button"
            >
              <Upload aria-hidden="true" size={16} />
              Upload a copy
            </button>
            <input
              accept="image/*"
              className="sr-only"
              onChange={(event) => {
                const chosen = event.target.files?.[0];
                if (chosen) onPlace(chosen);
                event.target.value = "";
              }}
              ref={file}
              tabIndex={-1}
              type="file"
            />
          </>
        )}
        <RemoveAction disabled={busy} onClick={onDiscard}>
          Let it go
        </RemoveAction>
      </div>
    </li>
  );
}
