import { ArrowUpRight, Mail } from "lucide-react";
import { CreatorMark } from "@/components/media/CreatorMark";
import type { Profile, ProfileLink } from "@/lib/api/query";
import styles from "./ProfilePreview.module.css";

export function ProfilePreview({
  handle,
  avatar,
  displayName,
  biography,
  contactEmail,
  links,
}: {
  handle: string;
  avatar: Profile["avatar"];
  displayName: string;
  biography: string;
  contactEmail: string;
  links: ProfileLink[];
}) {
  const shown = links.filter((link) => link.label || link.address);

  return (
    <aside className={styles.preview} aria-label="Profile preview">
      <p className={styles.caption}>As visitors see it</p>
      <div className={styles.band}>
        <CreatorMark handle={handle} portrait={avatar} compact />
        <div className={styles.identity}>
          <p className={styles.name}>{displayName || `@${handle}`}</p>
          {displayName ? <p className={styles.handle}>@{handle}</p> : null}
        </div>
        {biography ? <p className={styles.biography}>{biography}</p> : null}
        {contactEmail || shown.length > 0 ? (
          <ul className={styles.reach}>
            {contactEmail ? (
              <li>
                <Mail size={13} strokeWidth={1.6} aria-hidden="true" />
                {contactEmail}
              </li>
            ) : null}
            {shown.map((link, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: a link's position is its identity here
              <li key={index}>
                {link.label || link.address}
                <ArrowUpRight size={13} strokeWidth={1.6} aria-hidden="true" />
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <p className={styles.note}>
        Your published work follows straight after this, at /@{handle}.
      </p>
    </aside>
  );
}
