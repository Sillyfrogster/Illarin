import Image from "next/image";
import Link from "next/link";
import type { PostByline } from "@/lib/api/query";
import { bylineName, bylineProfile } from "@/lib/byline";
import styles from "./Byline.module.css";

/** Who wrote a post and what they were writing as, exactly as its byline was stored. */
export function Byline({ byline }: { byline: PostByline }) {
  const name = bylineName(byline);
  const profile = bylineProfile(byline);
  return (
    <div className={styles.byline}>
      <span className={styles.portrait} aria-hidden="true">
        {byline.avatar ? (
          <Image
            alt=""
            height={44}
            src={byline.avatar.url}
            unoptimized
            width={44}
          />
        ) : (
          <span className={styles.monogram}>
            {byline.handle.slice(0, 1).toUpperCase()}
          </span>
        )}
      </span>
      <span className={styles.who}>
        {profile ? (
          <a className={styles.name} href={profile}>
            {name}
          </a>
        ) : (
          <span className={styles.name}>{name}</span>
        )}
        <span className={styles.standing}>
          {byline.app ? (
            <>
              <Link
                className={styles.affiliation}
                href={`/blog/app/${byline.app.slug}`}
              >
                {byline.app.name}
              </Link>
              <span className={styles.publisher}>Published by Illarin</span>
            </>
          ) : (
            <>
              <span className={styles.affiliation}>Illarin Team</span>
              {standingOf(byline).length > 0 ? (
                <span className={styles.publisher}>
                  {standingOf(byline).join(" · ")}
                </span>
              ) : null}
            </>
          )}
        </span>
      </span>
    </div>
  );
}

/** The same attribution on one line, for a list where a portrait in every row is noise. */
export function BylineLine({
  byline,
  quiet,
}: {
  byline: PostByline;
  quiet?: boolean;
}) {
  const name = bylineName(byline);
  const profile = bylineProfile(byline);
  return (
    <span className={styles.line}>
      {profile ? (
        <a className={styles.lineName} href={profile}>
          {name}
        </a>
      ) : (
        <span className={styles.lineName}>{name}</span>
      )}
      {quiet ? null : byline.app ? (
        <Link className={styles.lineApp} href={`/blog/app/${byline.app.slug}`}>
          {byline.app.name}
        </Link>
      ) : (
        <span className={styles.lineApp}>Illarin Team</span>
      )}
    </span>
  );
}

function standingOf(byline: PostByline): string[] {
  return [...byline.positions, ...byline.distinctions];
}
