"use client";

import { motion, useReducedMotion } from "framer-motion";
import { ArrowLeft, ArrowRight, Maximize2 } from "lucide-react";
import Image, { type StaticImageData } from "next/image";
import { useState } from "react";
import { Button } from "./button";

export function PerspectiveCarousel({
  images,
}: {
  images: { id: string; src: StaticImageData; name?: string }[];
}) {
  const [active, setActive] = useState(0);
  const reduced = useReducedMotion();
  const select = (index: number) =>
    setActive(Math.min(Math.max(0, index), images.length - 1));
  if (!images.length)
    return <p className="vd:text-mute">No images in this gallery.</p>;
  return (
    <section
      aria-label="Image gallery"
      aria-roledescription="carousel"
      className="vd:overflow-hidden vd:rounded-plate vd:bg-deep vd:py-6"
    >
      <div
        className="vd:relative vd:h-80 vd:overflow-hidden"
        style={{ perspective: 1200 }}
      >
        <motion.div
          drag="x"
          dragConstraints={{ left: 0, right: 0 }}
          dragElastic={reduced ? 0 : 0.12}
          dragMomentum={false}
          onDragEnd={(_, info) => {
            if (Math.abs(info.offset.x) > 35)
              select(active + (info.offset.x < 0 ? 1 : -1));
          }}
          animate={{ x: -(active * 220 + 110) }}
          transition={
            reduced
              ? { duration: 0 }
              : { type: "spring", stiffness: 220, damping: 28 }
          }
          className="vd:absolute vd:top-0 vd:left-1/2 vd:flex vd:w-fit vd:touch-pan-y vd:items-center vd:cursor-grab vd:active:cursor-grabbing"
        >
          {images.map((item, index) => (
            <div
              key={item.id}
              className="vd:w-[220px] vd:shrink-0 vd:px-2"
              style={{ perspective: 1200 }}
            >
              <motion.div
                animate={{
                  rotateY: reduced ? 0 : (active - index) * 24,
                  scale: active === index ? 1 : 0.88,
                }}
                transition={
                  reduced
                    ? { duration: 0 }
                    : { type: "spring", stiffness: 220, damping: 28 }
                }
                style={{ transformStyle: "preserve-3d" }}
              >
                <button
                  type="button"
                  aria-label={`Show ${item.name ?? `image ${index + 1}`}`}
                  aria-current={active === index ? "true" : undefined}
                  tabIndex={active === index ? 0 : -1}
                  onClick={() => select(index)}
                  className="vd:block vd:h-72 vd:w-full vd:overflow-hidden vd:rounded-control vd:bg-plane"
                >
                  <Image
                    src={item.src}
                    alt={item.name ?? "Gallery image"}
                    draggable={false}
                    sizes="220px"
                    className="vd:h-full vd:w-full vd:object-contain"
                  />
                </button>
              </motion.div>
            </div>
          ))}
        </motion.div>
      </div>
      <div className="vd:flex vd:items-center vd:justify-between vd:gap-2 vd:px-4">
        <Button
          variant="secondary"
          size="icon"
          aria-label="Previous image"
          disabled={active === 0}
          onClick={() => select(active - 1)}
        >
          <ArrowLeft />
        </Button>
        <div
          aria-live="polite"
          className="vd:min-w-0 vd:text-center vd:text-meta"
        >
          <p className="vd:break-words">{images[active].name ?? "Image"}</p>
          <span className="vd:text-mute">
            {active + 1} / {images.length}
          </span>
        </div>
        <Button
          variant="secondary"
          size="icon"
          aria-label="Next image"
          disabled={active === images.length - 1}
          onClick={() => select(active + 1)}
        >
          <ArrowRight />
        </Button>
      </div>
      <a
        href={images[active].src.src}
        target="_blank"
        rel="noreferrer"
        className="vd:mx-auto vd:mt-2 vd:flex vd:min-h-11 vd:w-fit vd:items-center vd:gap-2 vd:px-3 vd:text-meta vd:text-mute"
      >
        View original image <Maximize2 className="vd:size-3.5" />
      </a>
    </section>
  );
}
