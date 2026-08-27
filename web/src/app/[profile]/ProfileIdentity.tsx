import { ArrowUpRight, Mail, SquarePen } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { CreatorMark } from "@/components/media/CreatorMark";
import type { Profile } from "@/lib/api/query";
import styles from "./ProfileIdentity.module.css";

export function ProfileIdentity({
  profile,
  isOwner,
}: {
  profile: Profile;
  isOwner: boolean;
}) {
  const name = profile.displayName || `@${profile.handle}`;
  const hasContactRow =
    Boolean(profile.contactEmail) || profile.links.length > 0;

  return (
    <header className={styles.band}>
      <div className={styles.art} aria-hidden="true" />
      <Shell className={styles.inner}>
        <CreatorMark handle={profile.handle} portrait={profile.avatar} />
        <div className={styles.identity}>
          <h1 data-long={name.length > 20 || undefined}>{name}</h1>
          {profile.displayName ? (
            <p className={styles.handle}>@{profile.handle}</p>
          ) : null}
          {profile.biography ? (
            <p className={styles.biography}>{profile.biography}</p>
          ) : null}
          {hasContactRow ? (
            <ul className={styles.reach}>
              {profile.contactEmail ? (
                <li>
                  <a href={`mailto:${profile.contactEmail}`}>
                    <Mail size={14} strokeWidth={1.6} aria-hidden="true" />
                    {profile.contactEmail}
                  </a>
                </li>
              ) : null}
              {profile.links.map((link) => (
                <li key={link.address}>
                  <a
                    href={link.address}
                    rel="nofollow ugc noopener"
                    target="_blank"
                  >
                    {link.label}
                    <ArrowUpRight
                      size={14}
                      strokeWidth={1.6}
                      aria-hidden="true"
                    />
                  </a>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        {isOwner ? (
          <Link className={styles.edit} href="/settings/profile">
            <SquarePen size={15} strokeWidth={1.6} aria-hidden="true" />
            Edit profile
          </Link>
        ) : null}
      </Shell>
    </header>
  );
}
