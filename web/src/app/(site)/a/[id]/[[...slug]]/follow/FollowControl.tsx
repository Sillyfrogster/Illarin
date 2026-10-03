"use client";

import { BellOff, BellPlus, BellRing } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/cn";
import { followWords } from "@/lib/work-follow";
import { useWorkFollowing } from "./state";

/** The Follow button beside the download button, which shows and changes whether the reader follows the work. */
export function FollowControl() {
  const following = useWorkFollowing();
  if (!following) return null;
  const words = followWords(following.follow, following.typeName);
  const Icon =
    following.follow.state === "stopped"
      ? BellOff
      : words.following
        ? BellRing
        : BellPlus;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          aria-label={words.name}
          className={cn(words.following && "text-accent hover:text-accent")}
        >
          <Icon aria-hidden="true" />
          {words.following ? "Following" : "Follow"}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        aria-label={`Updates to this ${following.typeName}`}
        className="w-[min(19rem,calc(100vw-2rem))]"
      >
        <p className="font-medium text-ink">
          {words.following ? "Following" : "Not following"}
        </p>
        <p className="mt-1 text-meta text-mute">{words.detail}</p>
        <Button
          className="mt-4 w-full"
          loading={following.pending}
          onClick={() => following.change(!words.following)}
          variant={words.following ? "secondary" : "primary"}
        >
          {words.action}
        </Button>
        {following.failure ? (
          <p className="mt-3 text-meta text-stop" role="alert">
            {following.failure}
          </p>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
