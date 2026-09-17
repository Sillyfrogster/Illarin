"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api/client";
import type { Profile } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { whatIsPublic } from "@/lib/profile-draft";

export function PublicProfileCard() {
  const { account } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const handle = account?.handle;

  useEffect(() => {
    if (!handle) return;
    let current = true;
    void (async () => {
      const { data: found } = await api<Profile>(
        "GET",
        `/v1/profiles/${handle}`,
        { cache: "no-store" },
      );
      if (found && current) setProfile(found);
    })();
    return () => {
      current = false;
    };
  }, [handle]);

  if (!account || !profile) return null;

  const showing = whatIsPublic({
    avatar: Boolean(profile.avatar),
    biography: profile.biography,
    contactEmail: profile.contactEmail,
    displayName: profile.displayName,
    links: profile.links,
  });

  return (
    <section className="flex flex-wrap items-center gap-x-4 gap-y-4 rounded-plate bg-inset px-5 py-5">
      <CreatorPortrait
        className="size-14"
        handle={profile.handle}
        picture={profile.avatar}
      />
      <div className="min-w-0 flex-1 basis-56">
        <h3 className="font-ui text-ui font-medium text-ink [overflow-wrap:anywhere]">
          {profile.displayName || `@${profile.handle}`}
        </h3>
        <p className="font-ui text-meta text-mute">
          {showing.length > 0
            ? `Showing ${showing.join(", ")}.`
            : "Only your handle is public so far."}
        </p>
      </div>
      <Button asChild className="w-full justify-between" variant="secondary">
        <Link href="/settings/profile">
          Edit profile
          <ArrowRight aria-hidden="true" />
        </Link>
      </Button>
    </section>
  );
}
