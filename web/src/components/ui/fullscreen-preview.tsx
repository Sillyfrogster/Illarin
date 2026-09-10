"use client";

import { X } from "lucide-react";
import Image from "next/image";
import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";

export type PreviewPicture = {
  src: string;
  alt: string;
  width: number;
  height: number;
};

export function FullscreenPreview({
  picture,
  onClose,
}: {
  picture: PreviewPicture;
  onClose: () => void;
}) {
  const close = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    close.current?.focus({ preventScroll: true });
    const leave = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    const held = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", leave);
    return () => {
      document.body.style.overflow = held;
      window.removeEventListener("keydown", leave);
    };
  }, [onClose]);

  return createPortal(
    <div
      aria-label={picture.alt || "Picture"}
      aria-modal="true"
      className="fixed inset-0 z-100 flex items-center justify-center bg-black/90 p-4"
      role="dialog"
    >
      <button
        aria-label="Close the picture"
        className="absolute top-4 right-4 flex size-11 items-center justify-center rounded-control text-over outline-offset-3 hover:bg-white/15"
        onClick={onClose}
        ref={close}
        type="button"
      >
        <X aria-hidden="true" className="size-5" />
      </button>
      <Image
        alt={picture.alt}
        className="max-h-full w-auto max-w-full object-contain"
        height={picture.height}
        sizes="100vw"
        src={picture.src}
        unoptimized
        width={picture.width}
      />
    </div>,
    document.body,
  );
}
