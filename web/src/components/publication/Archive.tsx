import Link from "next/link";
import type { PostArchive } from "@/lib/api/query";
import styles from "./Archive.module.css";
import { ArchiveLead, ArchiveRow } from "./ArchiveEntry";
import { PublicationArt } from "./PublicationArt";

export type ArchiveScope = {
  heading: string;
  address: string;
  narrowed: "category" | "app" | null;
  home: { label: string; href: string } | null;
};

/** The publication's front page: the newest post over the masthead art, then the rest. */
export function PublicationFront({ archive }: { archive: PostArchive }) {
  const [lead, ...rest] = archive.posts;
  if (!lead) return <EmptyArchive />;
  return (
    <div>
      <div className={styles.masthead}>
        <PublicationArt />
        <div className={styles.column}>
          <ArchiveLead post={lead} />
        </div>
      </div>
      <div className={styles.column}>
        <div className={styles.list}>
          {rest.map((post) => (
            <ArchiveRow key={post.id} post={post} />
          ))}
        </div>
        <ArchivePages archive={archive} address="/blog" />
      </div>
    </div>
  );
}

/** A numbered, category or app archive: a stated scope, then the same chronology. */
export function ScopedArchive({
  archive,
  scope,
}: {
  archive: PostArchive;
  scope: ArchiveScope;
}) {
  return (
    <div className={styles.column}>
      <header className={styles.scope}>
        <h1 className={styles.scopeHeading}>{scope.heading}</h1>
        <p className={styles.scopeMeta}>
          <span>
            {archive.total} {archive.total === 1 ? "post" : "posts"}
          </span>
          {scope.home ? (
            <a href={scope.home.href} rel="noreferrer noopener" target="_blank">
              {scope.home.label}
            </a>
          ) : null}
        </p>
      </header>
      {archive.posts.length > 0 ? (
        <div className={styles.list}>
          {archive.posts.map((post) => (
            <ArchiveRow key={post.id} narrowed={scope.narrowed} post={post} />
          ))}
        </div>
      ) : (
        <EmptyArchive />
      )}
      <ArchivePages archive={archive} address={scope.address} />
    </div>
  );
}

function ArchivePages({
  archive,
  address,
}: {
  archive: PostArchive;
  address: string;
}) {
  if (archive.pages < 2) return null;
  const newer = archive.page > 1;
  const older = archive.page < archive.pages;
  return (
    <nav aria-label="Archive pages" className={styles.pages}>
      {newer ? (
        <Link
          className={styles.step}
          href={pageAddress(address, archive.page - 1)}
          rel="prev"
        >
          Newer posts
        </Link>
      ) : (
        <span className={styles.spent}>Newer posts</span>
      )}
      <span className={styles.count}>
        Page {archive.page} of {archive.pages}
      </span>
      {older ? (
        <Link
          className={styles.step}
          href={pageAddress(address, archive.page + 1)}
          rel="next"
        >
          Older posts
        </Link>
      ) : (
        <span className={styles.spent}>Older posts</span>
      )}
    </nav>
  );
}

function EmptyArchive() {
  return (
    <p className={styles.empty}>Illarin has not published anything here yet.</p>
  );
}

/** Page one of any archive lives at the archive's own address, not under /page/1. */
export function pageAddress(address: string, page: number): string {
  return page === 1 ? address : `${address}/page/${page}`;
}
