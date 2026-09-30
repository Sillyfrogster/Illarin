"use client";

import { ImagePlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { addWorkImage } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useDraftedChanges } from "@/lib/drafted-changes";

/** CoverControl offers a display picture to every type of work. */
export function CoverControl({
  workId,
  hasCover,
  typeLabel,
}: {
  workId: string;
  hasCover: boolean;
  typeLabel: string;
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
        hidden
        onChange={(event) => send(event.target.files?.[0] ?? null)}
        ref={file}
        type="file"
      />
      <Button loading={sending} onClick={() => file.current?.click()}>
        {sending ? null : <ImagePlus aria-hidden="true" />}
        {hasCover ? "Replace the display picture" : "Add a display picture"}
      </Button>
      <p
        aria-live="polite"
        className={cn(
          "max-w-[34ch] text-center font-ui text-label",
          trouble ? "text-stop" : "text-mute",
        )}
      >
        {trouble ||
          `Shown for this ${typeLabel} in Browse and on shared links.`}
      </p>
    </div>
  );
}
