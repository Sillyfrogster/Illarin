import { ChevronLeft, ChevronRight, Rss } from "lucide-react";
import Link from "next/link";
import type { PostArchive } from "@/lib/api/query";
import {
  BLOG_DESCRIPTION,
  BLOG_TITLE,
  feedAddresses,
  pageAddress,
} from "@/lib/publication-metadata";
import styles from "./Archive.module.css";
import { ArchiveLead, ArchiveRow } from "./ArchiveEntry";
import { PublicationArt } from "./PublicationArt";

/** What an archive was narrowed by, which is also the word it shows a reader. */
export type ArchiveKind = "Publication" | "Category" | "Publication app";

export type ArchiveScope = {
  kind: ArchiveKind;
  heading: string;
  address: string;
  home: string | null;
};

const NARROWED = {
  Publication: null,
  Category: "category",
  "Publication app": "app",
} as const;

/** The publication's front page, leading with the newest post over the masthead art. */
export function PublicationFront({ archive }: { archive: PostArchive }) {
  const [lead, ...rest] = archive.posts;
  if (!lead) {
    return (
      <ScopedArchive
        archive={archive}
        scope={{
          kind: "Publication",
          heading: BLOG_TITLE,
          address: "/blog",
          home: null,
        }}
      />
    );
  }
  return (
    <div>
      <div className={styles.masthead}>
        <PublicationArt />
        <div className={styles.column}>
          <p className={styles.statement}>{BLOG_DESCRIPTION}</p>
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

/** A numbered, category or app archive, which states its scope above the same chronology. */
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
          <span className={styles.scopeKind}>{scope.kind}</span>
          <span>
            {archive.total} {archive.total === 1 ? "post" : "posts"}
          </span>
          {scope.home ? (
            <a href={scope.home} rel="noreferrer noopener" target="_blank">
              {siteName(scope.home)}
            </a>
          ) : null}
          <a className={styles.feed} href={feedAddresses(scope.address).rss}>
            <Rss aria-hidden="true" size={13} strokeWidth={1.9} />
            RSS feed
          </a>
        </p>
      </header>
      {archive.posts.length > 0 ? (
        <div className={styles.list}>
          {archive.posts.map((post) => (
            <ArchiveRow
              key={post.id}
              narrowed={NARROWED[scope.kind]}
              post={post}
            />
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
          <ChevronLeft aria-hidden="true" size={15} strokeWidth={1.9} />
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
          <ChevronRight aria-hidden="true" size={15} strokeWidth={1.9} />
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

/** The address a reader would recognise, without the scheme they never type. */
function siteName(address: string): string {
  try {
    return new URL(address).host.replace(/^www\./, "");
  } catch {
    return address;
  }
}
