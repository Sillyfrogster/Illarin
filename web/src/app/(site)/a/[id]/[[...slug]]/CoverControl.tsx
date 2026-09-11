"use client";

import { ImagePlus, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useRef, useState } from "react";
import { addAssetImage } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useWorkingCopy } from "@/lib/working-copy";

/** CoverControl offers a display picture to every kind of asset. */
export function CoverControl({
  assetId,
  hasCover,
  kindLabel,
}: {
  assetId: string;
  hasCover: boolean;
  kindLabel: string;
}) {
  const candidate = useWorkingCopy();
  const router = useRouter();
  const file = useRef<HTMLInputElement>(null);
  const [sending, setSending] = useState(false);
  const [trouble, setTrouble] = useState("");
  const field = useId();

  async function send(chosen: File | null) {
    if (!chosen || sending) return;
    setSending(true);
    setTrouble("");
    try {
      await addAssetImage(candidate, assetId, chosen, "avatar");
      router.refresh();
    } catch (error) {
      setTrouble(
        error instanceof Error
          ? error.message
          : "The picture could not be added. Try again.",
      );
    } finally {
      setSending(false);
      if (file.current) file.current.value = "";
    }
  }

  return (
    <div className="mt-3 flex flex-col items-center gap-2">
      <input
        accept="image/*"
        className="sr-only"
        id={field}
        onChange={(event) => send(event.target.files?.[0] ?? null)}
        ref={file}
        type="file"
      />
      <label
        className={cn(
          "inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-control bg-deep px-4 font-ui text-meta font-medium text-ink transition-colors duration-200 hover:bg-rule/45 has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent has-[:focus-visible]:outline-offset-3 motion-reduce:transition-none",
          sending && "pointer-events-none opacity-60",
        )}
        htmlFor={field}
      >
        {sending ? (
          <Loader2
            aria-hidden="true"
            className="size-4 animate-spin text-accent"
          />
        ) : (
          <ImagePlus aria-hidden="true" className="size-4 text-accent" />
        )}
        {hasCover ? "Replace the display picture" : "Add a display picture"}
      </label>
      <p
        aria-live="polite"
        className={cn(
          "max-w-[34ch] text-center font-ui text-label",
          trouble ? "text-stop" : "text-mute",
        )}
      >
        {trouble ||
          `It stands for this ${kindLabel} in the catalog and on shared links.`}
      </p>
    </div>
  );
}
