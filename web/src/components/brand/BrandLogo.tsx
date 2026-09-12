import Image from "next/image";
import { cn } from "@/lib/cn";

const LOGOS = {
  full: { light: "black", dark: "white" },
  accent: { light: "on-light", dark: "on-dark" },
} as const;

export function BrandLogo({
  className,
  tone = "full",
}: {
  className?: string;
  tone?: keyof typeof LOGOS;
}) {
  const colors = LOGOS[tone];

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
        height={144}
        loading="eager"
        src={`/brand/illarin-horizontal-${colors.light}.svg`}
        width={429}
      />
      <Image
        alt=""
        aria-hidden="true"
        className="invisible col-start-1 row-start-1 h-auto w-full dark:visible"
        height={144}
        loading="eager"
        src={`/brand/illarin-horizontal-${colors.dark}.svg`}
        width={429}
      />
    </span>
  );
}
