import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
  return (
    <Avatar className={cn(FRAME[size], className)}>
      {picture ? (
        <AvatarImage
          alt=""
          fetchPriority={priority ? "high" : undefined}
          height={picture.height}
          src={picture.url}
          width={picture.width}
        />
      ) : null}
      <AvatarFallback
        aria-hidden="true"
        className={cn(INITIAL[size], portraitGround(handle))}
        delayMs={picture ? 600 : 0}
      >
        {handle.slice(0, 1).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
}
