"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ChevronLeft, ChevronRight } from "lucide-react";
import Image from "next/image";
import { type KeyboardEvent, useState } from "react";
import { cn } from "@/lib/cn";

export type CarouselPicture = {
  id: string;
  src: string;
  width: number;
  height: number;
  name?: string;
};

/** How far the neighbouring pictures turn away from the one being looked at */
const ROTATION_STEP = 34;

const DEFAULT_SLIDE_WIDTH_PX = 224;

/** A row of pictures turned in space, with the chosen one facing the reader */
export function PerspectiveCarousel({
  pictures,
  label,
  slideWidth = DEFAULT_SLIDE_WIDTH_PX,
  className,
}: {
  pictures: CarouselPicture[];
  label: string;
  slideWidth?: number;
  className?: string;
}) {
  const [here, setHere] = useState(0);
  const reduced = useReducedMotion();
  const last = pictures.length - 1;
  const chosen = Math.min(here, last);
  const settle = reduced
    ? { duration: 0 }
    : ({ type: "spring", stiffness: 210, damping: 26 } as const);

  if (pictures.length === 0) return null;

  function show(index: number) {
    setHere(Math.min(Math.max(0, index), last));
  }

  function moveWithArrows(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      show(chosen - 1);
    }
    if (event.key === "ArrowRight") {
      event.preventDefault();
      show(chosen + 1);
    }
  }

  return (
    <section
      aria-label={label}
      aria-roledescription="carousel"
      className={cn("rounded-plate bg-deep py-5", className)}
    >
      <div className="relative h-72 overflow-hidden [perspective:1200px]">
        <motion.div
          animate={{ x: -(chosen * slideWidth + slideWidth / 2) }}
          className="absolute top-1/2 left-1/2 flex w-fit -translate-y-1/2 items-center"
          transition={settle}
        >
          {pictures.map((picture, index) => (
            <div
              className="shrink-0 px-2 [perspective:1200px]"
              key={picture.id}
              style={{ width: slideWidth }}
            >
              <motion.div
                animate={{
                  rotateY: reduced ? 0 : (chosen - index) * ROTATION_STEP,
                  scale: index === chosen ? 1 : 0.86,
                }}
                className="[transform-style:preserve-3d]"
                transition={settle}
              >
                <button
                  aria-current={index === chosen ? "true" : undefined}
                  aria-label={`Show ${picture.name || `picture ${index + 1}`}`}
                  className="block h-64 w-full cursor-pointer rounded-control outline-offset-3"
                  onClick={() => show(index)}
                  onKeyDown={moveWithArrows}
                  tabIndex={index === chosen ? 0 : -1}
                  type="button"
                >
                  <Image
                    alt={picture.name || ""}
                    className="h-full w-full rounded-control object-contain"
                    draggable={false}
                    height={picture.height}
                    sizes="300px"
                    src={picture.src}
                    unoptimized
                    width={picture.width}
                  />
                </button>
              </motion.div>
            </div>
          ))}
        </motion.div>
      </div>
      <div className="mt-3 flex items-center justify-between gap-2 px-3">
        <button
          aria-label="Show the picture before this one"
          className="flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-plane hover:text-ink disabled:opacity-35"
          disabled={chosen === 0}
          onClick={() => show(chosen - 1)}
          type="button"
        >
          <ChevronLeft aria-hidden="true" className="size-5" />
        </button>
        <p aria-live="polite" className="min-w-0 text-center text-meta">
          <span className="block break-words text-ink">
            {pictures[chosen].name || `Picture ${chosen + 1}`}
          </span>
          <span className="text-mute">
            {chosen + 1} of {pictures.length}
          </span>
        </p>
        <button
          aria-label="Show the picture after this one"
          className="flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-plane hover:text-ink disabled:opacity-35"
          disabled={chosen === last}
          onClick={() => show(chosen + 1)}
          type="button"
        >
          <ChevronRight aria-hidden="true" className="size-5" />
        </button>
      </div>
    </section>
  );
}
