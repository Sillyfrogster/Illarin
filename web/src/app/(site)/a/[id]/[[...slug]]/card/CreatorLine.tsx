import Image from "next/image";
import Link from "next/link";
import type { Profile } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";

function plural(count: number, word: string): string {
  return `${count.toLocaleString("en-US")} ${word}${count === 1 ? "" : "s"}`;
}

/** CreatorLine introduces the person who made the work: their picture, name and handle, and how much they have made. */
export function CreatorLine({
  handle,
  profile,
}: {
  handle: string;
  profile: Profile | null;
}) {
  const name = profile?.displayName || handle;
  return (
    <Link
      className={cn(
        "group/creator -m-2 flex min-w-0 items-center gap-3.5 rounded-plate p-2 transition-colors duration-150 ease-(--ease-wipe) hover:bg-fill",
        focusRing,
      )}
      href={`/@${handle}`}
    >
      <span className="grid size-12 shrink-0 place-items-center overflow-hidden rounded-full bg-fill">
        {profile?.avatar ? (
          <Image
            alt=""
            className="size-full object-cover"
            height={48}
            src={profile.avatar.url}
            unoptimized
            width={48}
          />
        ) : (
          <span className="font-display text-section leading-none text-mute">
            {handle.slice(0, 1).toUpperCase()}
          </span>
        )}
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-ui font-medium text-ink group-hover/creator:text-accent">
          {name}
        </span>
        <span className="truncate text-meta text-mute">
          @{handle}
          {profile
            ? ` · ${plural(profile.works, "work")} · ${plural(profile.followers, "follower")}`
            : null}
        </span>
      </span>
    </Link>
  );
}
