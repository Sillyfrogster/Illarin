"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { CreatorMark } from "@/components/media/CreatorMark";
import type { Profile } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import styles from "./PublicProfileCard.module.css";

export function PublicProfileCard() {
  const { account } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const handle = account?.handle;

  useEffect(() => {
    if (!handle) return;
    let current = true;
    void (async () => {
      const response = await fetch(`/api/v1/profiles/${handle}`, {
        cache: "no-store",
        credentials: "same-origin",
      });
      if (!response.ok || !current) return;
      const found = (await response.json()) as Profile;
      if (current) setProfile(found);
    })();
    return () => {
      current = false;
    };
  }, [handle]);

  if (!account || !profile) return null;

  const filled = [
    profile.displayName && "display name",
    profile.avatar && "avatar",
    profile.biography && "biography",
    profile.contactEmail && "contact",
    profile.links.length > 0 &&
      `${profile.links.length} ${profile.links.length === 1 ? "link" : "links"}`,
  ].filter(Boolean);

  return (
    <section className={styles.card}>
      <CreatorMark handle={profile.handle} portrait={profile.avatar} compact />
      <div className={styles.copy}>
        <h3>{profile.displayName || `@${profile.handle}`}</h3>
        <p>
          {filled.length > 0
            ? `Showing ${filled.join(", ")}.`
            : "Only your handle is public so far."}
        </p>
      </div>
      <Link className={styles.action} href="/settings/profile">
        Edit
        <ArrowRight size={15} strokeWidth={1.7} aria-hidden="true" />
      </Link>
    </section>
  );
}
