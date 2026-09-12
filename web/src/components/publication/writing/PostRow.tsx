"use client";

import { Clock, Eye, EyeOff, PenLine, Trash2 } from "lucide-react";
import Link from "next/link";
import type { Post } from "@/lib/api/query";
import { postPath } from "@/lib/blog-paths";
import { cn } from "@/lib/cn";
import { readableDate, shortMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import { useOrigins } from "@/lib/origins";
import {
  goingLiveAt,
  type Lifecycle,
  lifecycleName,
  lifecycleOf,
} from "@/lib/post-standing";

const MARKS = {
  draft: PenLine,
  published: Eye,
  withdrawn: EyeOff,
  deleted: Trash2,
} satisfies Record<Lifecycle, typeof PenLine>;

const CLOSING_SOON = 7 * 24 * 60 * 60 * 1000;

const TAG =
  "inline-flex min-h-7 items-center gap-1.5 rounded-control px-2.5 font-ui text-label font-medium whitespace-nowrap";

export function PostRow({ post }: { post: Post }) {
  const { blog } = useOrigins();
  const state = lifecycleOf(post);
  const Mark = MARKS[state];
  const going = goingLiveAt(post);

  return (
    <li className="group relative min-w-0 rounded-plate px-4 py-5 transition-colors duration-200 not-first:border-t not-first:border-rule/45 hover:border-transparent hover:bg-deep motion-reduce:transition-none sm:px-5">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-2">
        <h3 className="min-w-0 font-display text-section leading-snug font-medium text-ink wrap-anywhere">
          <Link
            className="text-ink outline-offset-3 before:absolute before:inset-0 before:content-[''] hover:text-accent"
            href={`/admin/blog/${post.id}`}
          >
            {post.title || "Untitled post"}
          </Link>
        </h3>
        <span
          className={cn(
            TAG,
            state === "published"
              ? "bg-accent-wash text-accent"
              : state === "deleted" || state === "withdrawn"
                ? "bg-stop-wash text-stop"
                : "bg-deep text-mute group-hover:bg-rule/45",
          )}
        >
          <Mark aria-hidden="true" className="size-3.5" strokeWidth={2} />
          {lifecycleName(state)}
        </span>
      </div>

      <p className="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 font-prose text-meta text-mute">
        <span>{post.category.label}</span>
        {post.app ? <span>{post.app.name}</span> : null}
        <span>{dateWords(post, state)}</span>
      </p>

      {going || post.deletion || state === "published" ? (
        <p className="relative mt-3 flex flex-wrap items-center gap-2">
          {going ? (
            <span className={cn(TAG, "bg-accent-wash text-accent")}>
              <Clock aria-hidden="true" className="size-3.5" strokeWidth={2} />
              Goes live {shortMoment(going)}
            </span>
          ) : null}
          {post.deletion ? <Deadline until={post.deletion.until} /> : null}
          {state === "published" ? (
            <a
              className="inline-flex min-h-11 items-center font-ui text-label font-medium text-accent underline-offset-4 outline-offset-3 hover:underline"
              href={new URL(postPath(post.slug), blog).href}
            >
              Read it on the blog
            </a>
          ) : null}
        </p>
      ) : null}
    </li>
  );
}

function Deadline({ until }: { until: string }) {
  const closing = new Date(until).getTime() - Date.now() < CLOSING_SOON;
  return (
    <span
      className={cn(
        TAG,
        closing ? "bg-stop-wash text-stop" : "bg-deep text-mute",
      )}
      suppressHydrationWarning
    >
      {remainingDeletionWindow(until)}
    </span>
  );
}

function dateWords(post: Post, state: Lifecycle): string {
  if (post.deletion) return `Deleted ${readableDate(post.deletion.at)}`;
  if (state === "published" || state === "withdrawn") {
    return `Published ${readableDate(post.publishedAt ?? post.createdAt)}`;
  }
  return `Started ${readableDate(post.createdAt)}`;
}
