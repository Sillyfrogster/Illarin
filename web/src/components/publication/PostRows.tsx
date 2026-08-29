"use client";

import { Plus } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { Section } from "@/components/console/Section";
import { startPost } from "@/lib/api/posts";
import type { Post, PublicationWorkspace } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./PostRows.module.css";

export function PostRows({
  posts,
  workspace,
  onFailure,
}: {
  posts: Post[];
  workspace: PublicationWorkspace;
  onFailure: (message: string) => void;
}) {
  const router = useRouter();
  const [starting, setStarting] = useState(false);
  const [busy, setBusy] = useState(false);
  const hasGrant = workspace.grants.length > 0;
  const [grantId, setGrantId] = useState(workspace.grants[0]?.id ?? "");
  const [categoryId, setCategoryId] = useState(
    workspace.categories[0]?.id ?? "",
  );
  const [title, setTitle] = useState("");

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
      count={posts.length}
      title="Posts"
    >
      {posts.length === 0 ? (
        <p className={styles.none}>
          Nothing written yet. A post starts as a private draft and stays that
          way until you publish it.
        </p>
      ) : (
        <ol className={rows.list}>
          {posts.map((post) => (
            <li className={rows.row} data-plain="true" key={post.id}>
              <span className={rows.name}>
                <Link href={`/admin/blog/${post.id}`}>{post.title}</Link>
              </span>
              <span className={rows.detail}>
                {post.category.label}
                {post.app ? ` · ${post.app.name}` : ""} ·{" "}
                {post.publishedAt
                  ? `Published ${readableDate(post.publishedAt)}`
                  : `Started ${readableDate(post.createdAt)}`}
              </span>
              <span className={rows.actions}>
                <span
                  className={styles.status}
                  data-live={post.status === "published" || undefined}
                >
                  {post.status === "published" ? "Published" : "Draft"}
                </span>
                {post.status === "published" ? (
                  <Link className={styles.read} href={`/blog/${post.slug}`}>
                    Read
                  </Link>
                ) : null}
              </span>
            </li>
          ))}
        </ol>
      )}

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
