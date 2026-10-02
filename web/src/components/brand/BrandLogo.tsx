import Image from "next/image";
import { cn } from "@/lib/cn";

const SHAPES = {
  wordmark: { file: "illarin-horizontal", width: 429, height: 144 },
  mark: { file: "illarin-mark", width: 118, height: 82 },
} as const;

/** BrandLogo is the violet butterfly, with the Illarin wordmark beside it unless only the mark is asked for. */
export function BrandLogo({
  className,
  shape = "wordmark",
}: {
  className?: string;
  shape?: keyof typeof SHAPES;
}) {
  const { file, width, height } = SHAPES[shape];

  return (
    <span
      aria-label="Illarin"
      className={cn("inline-grid w-32 shrink-0 align-middle", className)}
      role="img"
    >
      <Image
        alt=""
        aria-hidden="true"
        className="col-start-1 row-start-1 h-auto w-full dark:invisible"
        height={height}
        loading="eager"
        src={`/brand/${file}-on-light.svg`}
        width={width}
      />
      <Image
        alt=""
        aria-hidden="true"
        className="invisible col-start-1 row-start-1 h-auto w-full dark:visible"
        height={height}
        loading="eager"
        src={`/brand/${file}-on-dark.svg`}
        width={width}
      />
    </span>
  );
}
