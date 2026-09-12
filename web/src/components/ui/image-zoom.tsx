"use client";

import { Expand, X } from "lucide-react";
import type { ReactNode } from "react";
import Zoom from "react-medium-image-zoom";
import { cn } from "@/lib/cn";

const ZOOM_BUTTON =
  "[&_[data-rmiz-btn-zoom]]:absolute [&_[data-rmiz-btn-zoom]]:right-3 [&_[data-rmiz-btn-zoom]]:bottom-3 [&_[data-rmiz-btn-zoom]]:flex [&_[data-rmiz-btn-zoom]]:size-11 [&_[data-rmiz-btn-zoom]]:cursor-zoom-in [&_[data-rmiz-btn-zoom]]:items-center [&_[data-rmiz-btn-zoom]]:justify-center [&_[data-rmiz-btn-zoom]]:rounded-full [&_[data-rmiz-btn-zoom]]:border-0 [&_[data-rmiz-btn-zoom]]:bg-field [&_[data-rmiz-btn-zoom]]:p-0 [&_[data-rmiz-btn-zoom]]:text-ink [&_[data-rmiz-btn-zoom]]:shadow-cover [&_[data-rmiz-btn-zoom]]:outline-offset-3 [&_[data-rmiz-btn-zoom]]:transition-opacity [&_[data-rmiz-btn-zoom]]:duration-200 motion-reduce:[&_[data-rmiz-btn-zoom]]:transition-none";

const ZOOM_BUTTON_REVEAL =
  "lg:[&_[data-rmiz-btn-zoom]]:opacity-0 lg:[&:hover_[data-rmiz-btn-zoom]]:opacity-100 lg:[&_[data-rmiz-btn-zoom]:focus-visible]:opacity-100";

const DIALOG =
  "[&::backdrop]:hidden [&[open]]:fixed [&[open]]:inset-0 [&[open]]:m-0 [&[open]]:h-dvh [&[open]]:max-h-none [&[open]]:w-dvw [&[open]]:max-w-none [&[open]]:overflow-hidden [&[open]]:overscroll-none [&[open]]:border-0 [&[open]]:bg-transparent [&[open]]:p-0";

const OVERLAY =
  '[&_[data-rmiz-modal-overlay]]:absolute [&_[data-rmiz-modal-overlay]]:inset-0 [&_[data-rmiz-modal-overlay]]:transition-colors [&_[data-rmiz-modal-overlay]]:duration-300 [&_[data-rmiz-modal-overlay="hidden"]]:bg-transparent [&_[data-rmiz-modal-overlay="visible"]]:bg-media/90 motion-reduce:[&_[data-rmiz-modal-overlay]]:transition-none';

const ENLARGED =
  "[&_[data-rmiz-modal-content]]:relative [&_[data-rmiz-modal-content]]:size-full [&_[data-rmiz-modal-img]]:absolute [&_[data-rmiz-modal-img]]:origin-top-left [&_[data-rmiz-modal-img]]:cursor-zoom-out [&_[data-rmiz-modal-img]]:transition-transform [&_[data-rmiz-modal-img]]:duration-300 motion-reduce:[&_[data-rmiz-modal-img]]:transition-none";

const UNZOOM_BUTTON =
  "[&_[data-rmiz-btn-unzoom]]:absolute [&_[data-rmiz-btn-unzoom]]:top-4 [&_[data-rmiz-btn-unzoom]]:right-4 [&_[data-rmiz-btn-unzoom]]:z-1 [&_[data-rmiz-btn-unzoom]]:flex [&_[data-rmiz-btn-unzoom]]:size-11 [&_[data-rmiz-btn-unzoom]]:cursor-zoom-out [&_[data-rmiz-btn-unzoom]]:items-center [&_[data-rmiz-btn-unzoom]]:justify-center [&_[data-rmiz-btn-unzoom]]:rounded-control [&_[data-rmiz-btn-unzoom]]:border-0 [&_[data-rmiz-btn-unzoom]]:bg-transparent [&_[data-rmiz-btn-unzoom]]:p-0 [&_[data-rmiz-btn-unzoom]]:text-over [&_[data-rmiz-btn-unzoom]]:outline-offset-3 [&_[data-rmiz-btn-unzoom]:hover]:bg-over/15";

/** ImageZoom grows a picture from where it sits to fill the window, and shrinks it back. */
export function ImageZoom({
  alt,
  children,
  className,
  detailSrc,
  revealOnHover = false,
}: {
  alt: string;
  children: ReactNode;
  className?: string;
  detailSrc: string;
  revealOnHover?: boolean;
}) {
  return (
    <div
      className={cn(
        "relative [&_[data-rmiz-ghost]]:pointer-events-none [&_[data-rmiz-ghost]]:absolute [&_[data-rmiz-content=found]_img]:cursor-zoom-in [&_svg]:size-4",
        ZOOM_BUTTON,
        revealOnHover && ZOOM_BUTTON_REVEAL,
        className,
      )}
    >
      <Zoom
        a11yNameButtonUnzoom="Close the picture"
        a11yNameButtonZoom="See at full size"
        IconUnzoom={X}
        IconZoom={Expand}
        classDialog={cn(
          DIALOG,
          OVERLAY,
          ENLARGED,
          UNZOOM_BUTTON,
          "[&_svg]:size-5",
        )}
        zoomImg={{ alt, src: detailSrc }}
        zoomMargin={16}
      >
        {children}
      </Zoom>
    </div>
  );
}
