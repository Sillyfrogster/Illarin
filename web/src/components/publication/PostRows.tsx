"use client";

import { Clock, Eye, EyeOff, PenLine, Plus, Trash2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { Section } from "@/components/console/Section";
import { startPost } from "@/lib/api/posts";
import type { Post, PublicationWorkspace } from "@/lib/api/query";
import { readableDate, shortMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import {
  goingLiveAt,
  inStanding,
  type Lifecycle,
  lifecycleName,
  lifecycleOf,
  nothingThere,
  STANDINGS,
  type Standing,
} from "@/lib/post-standing";
import styles from "./PostRows.module.css";
import { PostStandings } from "./PostStandings";

const MARKS = {
  draft: PenLine,
  published: Eye,
  withdrawn: EyeOff,
  deleted: Trash2,
} satisfies Record<Lifecycle, typeof PenLine>;

/** How close a recovery deadline has to be before it is stated as a warning. */
const CLOSING_SOON = 7 * 24 * 60 * 60 * 1000;

export function PostRows({
  posts,
  deleted,
  workspace,
  onFailure,
}: {
  posts: Post[];
  deleted: Post[];
  workspace: PublicationWorkspace;
  onFailure: (message: string) => void;
}) {
  const router = useRouter();
  const [standing, setStanding] = useState<Standing>("everything");
  const [starting, setStarting] = useState(false);
  const [busy, setBusy] = useState(false);
  const hasGrant = workspace.grants.length > 0;
  const [grantId, setGrantId] = useState(workspace.grants[0]?.id ?? "");
  const [categoryId, setCategoryId] = useState(
    workspace.categories[0]?.id ?? "",
  );
  const [title, setTitle] = useState("");

  const counts = useMemo(() => tally(posts, deleted), [posts, deleted]);
  const shown = (standing === "deleted" ? deleted : posts).filter((post) =>
    inStanding(post, standing),
  );

  async function start() {
    setBusy(true);
    const answer = await startPost({
      grantId: grantId || undefined,
      categoryId,
      title,
    });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    router.push(`/admin/blog/${answer.value.id}`);
  }

  return (
    <Section
      action={
        <button
          className={styles.start}
          onClick={() => setStarting(true)}
          type="button"
        >
          <Plus size={15} strokeWidth={1.9} aria-hidden="true" />
          New post
        </button>
      }
      title="Posts"
    >
      <div className={styles.frame}>
        <PostStandings
          chosen={standing}
          counts={counts}
          onChoose={setStanding}
        />
        <div className={styles.results}>
          {shown.length === 0 ? (
            <p className={styles.none}>{nothingThere(standing)}</p>
          ) : (
            <ol className={rows.list}>
              {shown.map((post) => (
                <PostLine key={post.id} post={post} />
              ))}
            </ol>
          )}
        </div>
      </div>

      <FormDialog
        busy={busy}
        commit="Start writing"
        hint="You can change all of this while you write."
        onClose={() => setStarting(false)}
        onCommit={() => void start()}
        open={starting}
        ready={title.trim().length > 0 && categoryId !== ""}
        title="New post"
      >
        <Field htmlFor="new-post-title" label="Title">
          <input
            id="new-post-title"
            maxLength={160}
            onChange={(event) => setTitle(event.target.value)}
            value={title}
          />
        </Field>
        <Field htmlFor="new-post-category" label="Category">
          <select
            id="new-post-category"
            onChange={(event) => setCategoryId(event.target.value)}
            value={categoryId}
          >
            {workspace.categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.label}
              </option>
            ))}
          </select>
        </Field>
        {workspace.grants.length > 1 || (workspace.admin && hasGrant) ? (
          <Field
            hint={
              workspace.admin
                ? "An Illarin post carries the Illarin Team byline instead."
                : undefined
            }
            htmlFor="new-post-grant"
            label="Publish as"
          >
            <select
              id="new-post-grant"
              onChange={(event) => setGrantId(event.target.value)}
              value={grantId}
            >
              {workspace.grants.map((grant) => (
                <option key={grant.id} value={grant.id}>
                  {grant.app.name}
                </option>
              ))}
              {workspace.admin ? <option value="">Illarin</option> : null}
            </select>
          </Field>
        ) : null}
      </FormDialog>
    </Section>
  );
}

function PostLine({ post }: { post: Post }) {
  const state = lifecycleOf(post);
  const Mark = MARKS[state];
  const going = goingLiveAt(post);
  return (
    <li className={rows.row} data-name="wide" data-plain="true">
      <span className={rows.name}>
        <Link href={`/admin/blog/${post.id}`}>{post.title}</Link>
      </span>
      <span className={rows.detail}>
        {post.category.label}
        {post.app ? ` · ${post.app.name}` : ""} · {dateWords(post, state)}
      </span>
      <span className={rows.actions}>
        {post.deletion ? <Deadline until={post.deletion.until} /> : null}
        {going ? (
          <span className={styles.waiting}>
            <Clock size={13} strokeWidth={2} aria-hidden="true" />
            Goes live {shortMoment(going)}
          </span>
        ) : null}
        <span className={styles.mark} data-live={state === "published"}>
          <Mark size={13} strokeWidth={2} aria-hidden="true" />
          {lifecycleName(state)}
        </span>
        {state === "published" ? (
          <Link className={styles.read} href={`/blog/${post.slug}`}>
            Read
          </Link>
        ) : null}
      </span>
    </li>
  );
}

function Deadline({ until }: { until: string }) {
  const closing = new Date(until).getTime() - Date.now() < CLOSING_SOON;
  return (
    <span
      className={styles.deadline}
      data-closing={closing || undefined}
      suppressHydrationWarning
    >
      {remainingDeletionWindow(until)}
    </span>
  );
}

/** The date that matters for the state a post is in. */
function dateWords(post: Post, state: Lifecycle): string {
  if (post.deletion) return `Deleted ${readableDate(post.deletion.at)}`;
  if (state === "published" || state === "withdrawn") {
    return `Published ${readableDate(post.publishedAt ?? post.createdAt)}`;
  }
  return `Started ${readableDate(post.createdAt)}`;
}

/** What each standing holds, so the rail says so before it is opened. */
function tally(posts: Post[], deleted: Post[]): Record<Standing, number> {
  const counted = {} as Record<Standing, number>;
  for (const standing of STANDINGS) {
    const from = standing === "deleted" ? deleted : posts;
    counted[standing] = from.filter((post) =>
      inStanding(post, standing),
    ).length;
  }
  return counted;
}
