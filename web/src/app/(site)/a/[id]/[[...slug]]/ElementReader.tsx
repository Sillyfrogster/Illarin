"use client";

import { ChevronUp } from "lucide-react";
import { useEffect, useRef } from "react";
import { FormattingNotice } from "@/components/ui/RichText";
import type { AssetElement, AssetImage } from "@/lib/api/query";
import { formattingWasRemoved, richTextsOf } from "@/lib/rich-text";
import { ElementContent } from "./ElementBody";

export function ElementReader({
  element,
  images,
  onDismiss,
}: {
  element: AssetElement;
  images: AssetImage[];
  onDismiss: () => void;
}) {
  const dismiss = useRef<HTMLButtonElement>(null);
  const titleId = `element-reader-${element.id}`;

  useEffect(() => {
    dismiss.current?.focus({ preventScroll: true });
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      onDismiss();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onDismiss]);

  return (
    <section
      aria-labelledby={titleId}
      className="mt-6 rounded-plate bg-deep"
      data-measurement-ignore
    >
      <header className="flex flex-col items-start justify-between gap-4 p-4 sm:flex-row sm:items-center">
        <h3
          className="min-w-0 font-display text-section font-medium tracking-tight text-ink [overflow-wrap:anywhere]"
          id={titleId}
        >
          {element.label || "Content"}
        </h3>
        <button
          className="inline-flex min-h-11 w-full shrink-0 items-center justify-center gap-2 rounded-control bg-plane px-3 text-meta font-medium text-mute outline-offset-3 hover:text-ink sm:w-auto"
          onClick={onDismiss}
          ref={dismiss}
          type="button"
        >
          <ChevronUp aria-hidden="true" size={16} />
          Collapse
        </button>
      </header>
      <div className="max-h-[min(72dvh,760px)] overflow-y-auto px-4 pt-1 pb-6">
        <div className="flex min-w-0 flex-col gap-2.5 text-mute">
          <ElementContent element={element} images={images} />
          {formattingWasRemoved(richTextsOf(element)) ? (
            <FormattingNotice />
          ) : null}
        </div>
      </div>
      <p className="border-rule border-t px-4 py-3 text-meta text-mute">
        Esc closes this reader.
      </p>
    </section>
  );
}
