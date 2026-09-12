"use client";

import Image from "next/image";
import { ImageZoom } from "@/components/ui/image-zoom";
import { cn } from "@/lib/cn";
import {
  MOSAIC_GAP_PX,
  type MosaicPicture,
  mosaicRows,
  pictureWidth,
} from "@/lib/mosaic";
import { useMeasuredWidth } from "@/lib/use-measured-width";

/** Mosaic shows every picture at its own shape, justified row by row. */
export function Mosaic({
  className,
  label,
  pictures,
  rowHeight,
}: {
  className?: string;
  label: string;
  pictures: MosaicPicture[];
  rowHeight: number;
}) {
  const [measure, width] = useMeasuredWidth<HTMLDivElement>();

  if (pictures.length === 0) return null;

  const rows = mosaicRows(pictures, { rowHeight, width: width ?? 0 });

  return (
    <div className={cn("min-w-0", className)} ref={measure}>
      <ul
        aria-label={label}
        className="flex list-none flex-col"
        style={{ gap: MOSAIC_GAP_PX }}
      >
        {rows.map((row) => (
          <li
            className="flex min-w-0"
            key={row.pictures[0].id}
            style={{ gap: MOSAIC_GAP_PX }}
          >
            {row.pictures.map((picture) => (
              <figure
                className={cn("min-w-0", row.justified && "grow")}
                key={picture.id}
                style={{ flexBasis: pictureWidth(picture, row.height) }}
              >
                <ImageZoom
                  alt={picture.name || ""}
                  className="group block w-full overflow-hidden rounded-control bg-media"
                  detailSrc={picture.src}
                  revealOnHover
                >
                  <Image
                    alt={picture.name || ""}
                    className="w-full object-cover transition-transform duration-500 group-hover:scale-[1.04] motion-reduce:transform-none motion-reduce:transition-none"
                    draggable={false}
                    height={picture.height}
                    sizes="(max-width: 768px) 90vw, 460px"
                    src={picture.src}
                    style={{ height: row.height }}
                    unoptimized
                    width={picture.width}
                  />
                </ImageZoom>
                {picture.name ? (
                  <figcaption className="mt-1.5 truncate font-ui text-label text-mute">
                    {picture.name}
                  </figcaption>
                ) : null}
              </figure>
            ))}
          </li>
        ))}
      </ul>
    </div>
  );
}
