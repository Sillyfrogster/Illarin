import Image from "next/image";
import type { Profile } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { portraitGround } from "@/lib/portrait-tone";

type Size = "sm" | "md" | "lg";

const FRAME: Record<Size, string> = {
  lg: "size-24 sm:size-32 lg:size-40",
  md: "size-20",
  sm: "size-11",
};

const INITIAL: Record<Size, string> = {
  lg: "text-[clamp(2rem,5vw,3.25rem)]",
  md: "text-title",
  sm: "text-ui",
};

/**
 * A creator as a page shows them: their own picture, or the first letter of
 * their handle on one of a few grounds. The same plate serves the profile, the
 * account page and the preview beside the editor, so a creator sees the face
 * visitors meet wherever they look at it.
 */
export function CreatorPortrait({
  className,
  handle,
  picture,
  priority = false,
  size = "md",
}: {
  className?: string;
  handle: string;
  picture?: Profile["avatar"];
  priority?: boolean;
  size?: Size;
}) {
  const frame = cn(
    "block shrink-0 overflow-hidden rounded-plate",
    FRAME[size],
    className,
  );

  if (picture) {
    return (
      <span className={cn(frame, "bg-deep")}>
        <Image
          alt=""
          className="size-full object-cover"
          height={picture.height}
          priority={priority}
          src={picture.url}
          unoptimized
          width={picture.width}
        />
      </span>
    );
  }

  return (
    <span
      aria-hidden="true"
      className={cn(frame, "grid place-items-center", portraitGround(handle))}
    >
      <span className={cn("font-display font-medium", INITIAL[size])}>
        {handle.slice(0, 1).toUpperCase()}
      </span>
    </span>
  );
}
