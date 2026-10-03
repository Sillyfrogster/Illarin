"use client";

import { X } from "lucide-react";
import { type KeyboardEvent, useState } from "react";
import { useFieldControl } from "@/components/ui/field";
import { fieldSurface } from "@/components/ui/input";
import { cn, focusRing } from "@/lib/cn";

/** TagField holds tags as chips inside one filled field: Enter or a comma adds what is typed, and Backspace on an empty field takes the last one back. */
export function TagField({
  tags,
  onAdd,
  onRemove,
  id,
  className,
  "aria-describedby": describedBy,
  "aria-invalid": invalid,
}: {
  tags: readonly string[];
  onAdd: (tag: string) => boolean;
  onRemove: (index: number) => void;
  id?: string;
  className?: string;
  "aria-describedby"?: string;
  "aria-invalid"?: boolean;
}) {
  const field = useFieldControl({
    id,
    "aria-describedby": describedBy,
    "aria-invalid": invalid,
  });
  const [text, setText] = useState("");

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      if (text.trim() && onAdd(text.trim())) setText("");
      return;
    }
    if (event.key === "Backspace" && text === "" && tags.length > 0) {
      event.preventDefault();
      setText(tags[tags.length - 1]);
      onRemove(tags.length - 1);
    }
  }

  return (
    <div
      className={cn(
        fieldSurface,
        "flex min-h-control w-full min-w-0 flex-wrap items-center gap-1 p-1 has-[input:focus-visible]:border-accent has-[[aria-invalid=true]]:border-stop",
        className,
      )}
    >
      {tags.map((tag, index) => (
        <span
          className="inline-flex h-control-compact max-w-full items-center gap-0.5 rounded-chip bg-field pl-2 text-meta text-ink"
          key={tag}
        >
          <span className="truncate">{tag}</span>
          <button
            aria-label={`Remove ${tag}`}
            className={cn(
              "inline-grid aspect-square h-full shrink-0 cursor-pointer place-items-center rounded-chip text-mute transition-colors duration-80 hover:text-ink",
              focusRing,
            )}
            onClick={() => onRemove(index)}
            type="button"
          >
            <X aria-hidden="true" className="size-3.5" />
          </button>
        </span>
      ))}
      <input
        {...field}
        className="h-control-compact min-w-24 flex-1 bg-transparent px-2 outline-none placeholder:text-mute focus-visible:outline-none"
        enterKeyHint="done"
        onChange={(event) => setText(event.target.value)}
        onKeyDown={onKeyDown}
        placeholder={tags.length ? "" : "Add a tag"}
        value={text}
      />
    </div>
  );
}
