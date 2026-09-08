import { ArrowLeft, ExternalLink, Rss } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import type { PostArchive } from "@/lib/api/query";
import type { ArchiveNarrowing } from "@/lib/archive-entry";
import type { PostCover } from "@/lib/post-cover";
import {
  BLOG_DESCRIPTION,
  BLOG_HEADING,
  BLOG_HOME,
  feedAddresses,
} from "@/lib/publication-metadata";
import { ArchiveLead } from "./ArchiveEntry";
import { ArchiveList } from "./ArchiveList";
import { ArchivePages } from "./ArchivePages";
import { BackToIllarin } from "./BackToIllarin";

/** What an archive was narrowed by, which is also the word it shows a reader. */
export type ArchiveKind = "Publication" | "Category" | "Publication app";

export type ArchiveScope = {
  kind: ArchiveKind;
  heading: string;
  statement: string;
  address: string;
  home: string | null;
};

const NARROWED: Record<ArchiveKind, ArchiveNarrowing> = {
  Publication: null,
  Category: "category",
  "Publication app": "app",
};

/** The publication's front page, which leads with its newest writing and its picture. */
export function PublicationFront({
  archive,
  cover,
}: {
  archive: PostArchive;
  cover: PostCover | null;
}) {
  const [lead, ...rest] = archive.posts;
  const scope: ArchiveScope = {
    kind: "Publication",
    heading: BLOG_HEADING,
    statement: BLOG_DESCRIPTION,
    address: BLOG_HOME,
    home: null,
  };
  if (!lead) return <ScopedArchive archive={archive} scope={scope} />;
  return (
    <ArchivePage archive={archive} scope={scope}>
      <ArchiveLead cover={cover} post={lead} />
      {rest.length > 0 ? (
        <section className="mt-14">
          <h2 className="mb-1 font-display text-section font-medium">
            Earlier
          </h2>
          <ArchiveList narrowed={null} posts={rest} />
        </section>
      ) : null}
    </ArchivePage>
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
    <ArchivePage archive={archive} scope={scope}>
      {archive.posts.length > 0 ? (
        <ArchiveList narrowed={NARROWED[scope.kind]} posts={archive.posts} />
      ) : (
        <p className="max-w-[44ch] pb-section font-prose text-lede text-mute">
          Illarin has not published anything here yet.
        </p>
      )}
    </ArchivePage>
  );
}

function ArchivePage({
  archive,
  children,
  scope,
}: {
  archive: PostArchive;
  children: ReactNode;
  scope: ArchiveScope;
}) {
  return (
    <div className="mx-auto w-full max-w-[76rem] px-[var(--gutter)] pb-4">
      <div className="flex flex-wrap items-end justify-between gap-x-12 gap-y-4 pt-12">
        <h1 className="max-w-[18ch] font-display text-display font-medium tracking-[-0.035em] break-words">
          {scope.heading}
          <span aria-hidden="true" className="text-accent">
            .
          </span>
        </h1>
        <p className="max-w-[34ch] font-prose text-ui leading-6 text-mute">
          {scope.statement}
        </p>
      </div>
      <ArchiveFacts archive={archive} scope={scope} />
      {children}
      <ArchivePages
        address={scope.address}
        page={archive.page}
        pages={archive.pages}
      />
      <BackToIllarin />
    </div>
  );
}

/** What this archive is, how much of it there is, and the ways out of it. */
function ArchiveFacts({
  archive,
  scope,
}: {
  archive: PostArchive;
  scope: ArchiveScope;
}) {
  return (
    <div className="mt-2 mb-10 flex flex-wrap items-center gap-x-6 gap-y-1 text-meta">
      <span className="font-medium text-accent">{scope.kind}</span>
      <span className="text-mute">
        {archive.total} {archive.total === 1 ? "post" : "posts"}
      </span>
      {scope.kind === "Publication" ? null : (
        <Link
          className="flex min-h-11 items-center gap-2 text-mute hover:text-ink"
          href={BLOG_HOME}
        >
          <ArrowLeft aria-hidden="true" className="size-4" />
          All posts
        </Link>
      )}
      {scope.home ? (
        <a
          className="flex min-h-11 items-center gap-2 text-mute hover:text-ink"
          href={scope.home}
          rel="noreferrer noopener"
          target="_blank"
        >
          <ExternalLink aria-hidden="true" className="size-4" />
          {siteName(scope.home)}
        </a>
      ) : null}
      <a
        className="flex min-h-11 items-center gap-2 text-mute hover:text-ink"
        href={feedAddresses(scope.address).rss}
      >
        <Rss aria-hidden="true" className="size-4" />
        Follow the feed
      </a>
    </div>
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
