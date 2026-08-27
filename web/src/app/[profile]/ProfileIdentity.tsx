import { ArrowUpRight, Award, Mail, ShieldOff, SquarePen } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { Avatar } from "@/components/media/Avatar";
import type { Profile, ProfileDistinction } from "@/lib/api/query";
import styles from "./ProfileIdentity.module.css";
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
              Illarin has restricted this profile. Its published work is below.
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
          {profile.titles.length > 0 ? (
            <p className={styles.titles}>
              {profile.titles.map((title) => title.name).join(" · ")}
            </p>
          ) : null}
          {profile.badges.length > 0 ? (
            <ul className={styles.badges}>
              {profile.badges.map((badge) => (
                <BadgeMark key={badge.id} badge={badge} />
              ))}
            </ul>
          ) : null}
        </Shell>
      ) : null}
    </header>
  );
}

function BadgeMark({ badge }: { badge: ProfileDistinction }) {
  return (
    <li className={styles.badge} title={badge.explanation || undefined}>
      {badge.mark ? (
        <Image
          className={styles.badgeMark}
          src={badge.mark.url}
          alt=""
          width={22}
          height={22}
          unoptimized
        />
      ) : (
        <span className={styles.badgeMark} data-blank="true">
          <Award size={15} strokeWidth={1.7} aria-hidden="true" />
        </span>
      )}
      {badge.name}
      {badge.explanation ? (
        <span className="sr-only">. {badge.explanation}</span>
      ) : null}
    </li>
  );
}
