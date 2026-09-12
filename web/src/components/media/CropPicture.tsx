"use client";

import { RotateCcw } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  ImageCrop,
  ImageCropContent,
  useImageCrop,
} from "@/components/ui/image-crop";

/** Files the crop cannot keep intact, which upload as they are. */
export function cropsCleanly(file: File): boolean {
  return file.type !== "image/gif" && file.type.startsWith("image/");
}

/** CropPicture lets a person frame a chosen picture before it uploads. */
export function CropPicture({
  aspect,
  file,
  onCancel,
  onDone,
  title,
}: {
  aspect: number;
  file: File | null;
  onCancel: () => void;
  onDone: (file: File) => void;
  title: string;
}) {
  return (
    <Dialog
      onOpenChange={(open) => (open ? null : onCancel())}
      open={file !== null}
    >
      <DialogContent className="max-w-[44rem]">
        {file ? (
          <ImageCrop aspect={aspect} file={file}>
            <CropBody
              onCancel={onCancel}
              onDone={onDone}
              original={file}
              title={title}
            />
          </ImageCrop>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function CropBody({
  onCancel,
  onDone,
  original,
  title,
}: {
  onCancel: () => void;
  onDone: (file: File) => void;
  original: File;
  title: string;
}) {
  const { cut, reset } = useImageCrop();
  const [busy, setBusy] = useState(false);
  const [trouble, setTrouble] = useState("");

  async function apply() {
    setBusy(true);
    setTrouble("");
    try {
      onDone(await cut());
    } catch (error) {
      setTrouble(
        error instanceof Error
          ? error.message
          : "The picture could not be cropped.",
      );
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-0 flex-col gap-5 p-5 sm:p-6">
      <div className="pr-12">
        <DialogTitle className="font-display text-section font-medium text-ink">
          {title}
        </DialogTitle>
        <DialogDescription className="mt-1 font-ui text-meta text-mute">
          Drag the frame to choose what shows. Corners resize it.
        </DialogDescription>
      </div>
      <div className="grid min-h-0 place-items-center overflow-hidden rounded-control bg-media p-3">
        <ImageCropContent />
      </div>
      {trouble ? (
        <p className="font-ui text-meta text-stop" role="alert">
          {trouble}
        </p>
      ) : null}
      <div className="flex flex-wrap items-center gap-2">
        <Button loading={busy} onClick={() => void apply()} variant="primary">
          Use this crop
        </Button>
        <Button disabled={busy} onClick={() => onDone(original)}>
          Use the whole picture
        </Button>
        <Button disabled={busy} onClick={reset} variant="ghost">
          <RotateCcw aria-hidden="true" />
          Start again
        </Button>
        <Button
          className="ml-auto"
          disabled={busy}
          onClick={onCancel}
          variant="ghost"
        >
          Cancel
        </Button>
      </div>
    </div>
  );
}
