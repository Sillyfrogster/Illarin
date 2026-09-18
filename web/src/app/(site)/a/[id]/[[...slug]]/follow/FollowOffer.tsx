"use client";

import { BellPlus, BellRing } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { offersFollow } from "@/lib/work-follow";
import { useWorkFollowing } from "./state";

/** Asks, after a download or send, whether to hear when the work updates. */
export function FollowOffer() {
  const following = useWorkFollowing();
  const [asked, setAsked] = useState(false);
  if (!following) return null;

  if (asked && following.follow.state === "following") {
    return (
      <output className="mt-4 flex items-start gap-2 text-meta text-mute">
        <BellRing
          aria-hidden="true"
          className="mt-0.5 size-3.5 shrink-0 text-accent"
        />
        Following. You get a notification when this {following.typeName}{" "}
        updates.
      </output>
    );
  }
  if (!offersFollow(following.follow, following.notNow)) return null;

  return (
    <div className="mt-4">
      <p className="flex items-start gap-2 text-meta text-ink">
        <BellPlus
          aria-hidden="true"
          className="mt-0.5 size-3.5 shrink-0 text-accent"
        />
        Get a notification when this {following.typeName} updates?
      </p>
      <div className="mt-2 flex flex-wrap gap-2 pl-5.5">
        <Button
          loading={following.pending}
          onClick={() => {
            setAsked(true);
            following.change(true);
          }}
          size="compact"
        >
          Follow
        </Button>
        <Button
          disabled={following.pending}
          onClick={following.sayNotNow}
          size="compact"
          variant="ghost"
        >
          Not now
        </Button>
      </div>
      {asked && following.failure ? (
        <p className="mt-2 text-meta text-stop" role="alert">
          {following.failure}
        </p>
      ) : null}
    </div>
  );
}
