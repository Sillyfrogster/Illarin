"use client";

import { ImagePlus } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { addWorkImage } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useDraftedChanges } from "@/lib/drafted-changes";
import { FileMark } from "./workspace/fields";

/** The smallest cover that stays sharp at twice the pixel density on the work page and in Browse. */
const COVER_SIZE = "750 × 1000";

/** CoverControl adds or replaces a work's cover and shows the 3:4 crop Browse gives it. */
export function CoverControl({
  inFile,
  preview,
  workId,
}: {
  inFile: boolean;
  preview?: string;
  workId: string;
}) {
  const candidate = useDraftedChanges();
  const router = useRouter();
  const file = useRef<HTMLInputElement>(null);
  const [sending, setSending] = useState(false);
  const [trouble, setTrouble] = useState("");

  async function send(chosen: File | null) {
    if (!chosen || sending) return;
    setSending(true);
    setTrouble("");
    try {
      await addWorkImage(candidate, workId, chosen, "avatar");
      router.refresh();
    } catch (error) {
      setTrouble(
        error instanceof Error
          ? error.message
          : "The cover could not be added. Try again.",
      );
    } finally {
      setSending(false);
      if (file.current) file.current.value = "";
    }
  }

  return (
    <div className="mt-4 flex items-start gap-4">
      {preview ? (
        <figure className="w-16 shrink-0">
          <div className="relative aspect-3/4 overflow-hidden rounded-control bg-inset">
            <Image
              alt=""
              className="object-cover object-top"
              fill
              sizes="64px"
              src={preview}
              unoptimized
            />
          </div>
          <figcaption className="mt-1 text-center font-ui text-label text-mute">
            In Browse
          </figcaption>
        </figure>
      ) : null}
      <div className="flex min-w-0 flex-col items-start gap-2">
        <input
          accept="image/*"
          hidden
          onChange={(event) => send(event.target.files?.[0] ?? null)}
          ref={file}
          type="file"
        />
        <Button loading={sending} onClick={() => file.current?.click()}>
          {sending ? null : <ImagePlus aria-hidden="true" />}
          {preview ? "Replace cover" : "Add cover"}
        </Button>
        {inFile ? <FileMark /> : null}
        <p
          aria-live="polite"
          className={cn(
            "font-ui text-label",
            trouble ? "text-stop" : "text-mute",
          )}
        >
          {trouble || `Browse crops it to 3:4. Use at least ${COVER_SIZE}.`}
        </p>
      </div>
    </div>
  );
}
