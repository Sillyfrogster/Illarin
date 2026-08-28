import { ArrowUpRight, Mail, ShieldOff, SquarePen } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { Avatar } from "@/components/media/Avatar";
import type { Profile } from "@/lib/api/query";
import styles from "./ProfileIdentity.module.css";
import { ProfileRecognition } from "./ProfileRecognition";
import { RestrictionControl } from "./RestrictionControl";

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
  const hasStanding = profile.titles.length > 0 || profile.badges.length > 0;

  const actions = (
    <div className={styles.actions}>
      {isOwner && !profile.restricted ? (
        <Link className={styles.edit} href="/settings/profile">
          <SquarePen size={15} strokeWidth={1.6} aria-hidden="true" />
          Edit profile
        </Link>
      ) : null}
      <RestrictionControl
        handle={profile.handle}
        restricted={profile.restricted}
      />
    </div>
  );

  if (profile.restricted) {
    return (
      <header className={styles.band}>
        <div className={styles.art} aria-hidden="true" />
        <Shell className={styles.inner}>
          <Avatar handle={profile.handle} size="lg" />
          <div className={styles.identity}>
            <h1 data-long={profile.handle.length > 19 || undefined}>
              @{profile.handle}
            </h1>
            <p className={styles.restricted}>
              <ShieldOff size={15} strokeWidth={1.6} aria-hidden="true" />
              Illarin has hidden what this creator added to their profile. Their
              published work is below.
            </p>
          </div>
          {actions}
        </Shell>
      </header>
    );
  }

  return (
    <header className={styles.band}>
      <div className={styles.art} aria-hidden="true" />
      <Shell className={styles.inner}>
        <Avatar handle={profile.handle} portrait={profile.avatar} size="lg" />
        <div className={styles.identity}>
          <h1 data-long={name.length > 20 || undefined}>{name}</h1>
          <p className={styles.handle}>@{profile.handle}</p>
          {profile.positions.length > 0 ? (
            <p className={styles.positions}>
              {profile.positions.map((position) => position.name).join(" · ")}
            </p>
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
        {actions}
      </Shell>
      {hasStanding ? (
        <Shell className={styles.standing}>
          <h2 className={styles.standingLabel}>Given by Illarin</h2>
          <ProfileRecognition titles={profile.titles} badges={profile.badges} />
        </Shell>
      ) : null}
    </header>
  );
}
