"use client";

import { ArrowLeft, Eye, PenLine } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { ConsoleGate } from "@/components/console/ConsoleGate";
import { FormDialog } from "@/components/console/FormDialog";
import { ArticleIdentity } from "@/components/publication/ArticleIdentity";
import { PostBody } from "@/components/publication/PostBody";
import { publishPost, readPost, saveWorkingCopy } from "@/lib/api/posts";
import { readWorkspace } from "@/lib/api/publication";
import type {
  Post,
  PublicationApp,
  PublicationCategory,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { asPostDocument, type PostDocument } from "@/lib/post-document";
import { BodyEditor } from "./BodyEditor";
import { GrowingText } from "./GrowingText";
import { PostDetails } from "./PostDetails";
import styles from "./PostWriter.module.css";

/** How long the writer pauses before the working copy is saved. */
const AUTOSAVE_PAUSE = 1200;

type Draft = {
  categoryId: string;
  title: string;
  summary: string;
  slug: string;
  document: PostDocument;
  release: { appId: string; version: string; address: string } | null;
};

type Saving = "clean" | "dirty" | "saving" | "saved" | "conflict" | "refused";

export function PostWriter({ id }: { id: string }) {
  const { account } = useAuth();
  const [post, setPost] = useState<Post | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [apps, setApps] = useState<PublicationApp[]>([]);
  const [state, setState] = useState<Saving>("clean");
  const [refusal, setRefusal] = useState("");
  const [failure, setFailure] = useState("");
  const [previewing, setPreviewing] = useState(false);
  const [asking, setAsking] = useState(false);
  const [edition, setEdition] = useState(0);
  const version = useRef(0);

  const load = useCallback(async () => {
    const answer = await readPost(id);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setPost(answer.value);
    setDraft(asDraft(answer.value));
    version.current = answer.value.version;
    setState("clean");
    setRefusal("");
    setEdition((count) => count + 1);
  }, [id]);

  useEffect(() => {
    if (!account) return;
    void load();
    void readWorkspace().then((answer) => {
      if (!answer.value) return;
      setCategories(answer.value.categories);
      setApps(answer.value.apps);
    });
  }, [account, load]);

  const save = useCallback(async () => {
    if (!draft) return;
    setState("saving");
    const answer = await saveWorkingCopy(id, {
      version: version.current,
      ...draft,
      release: draft.release,
    });
    if (answer.value) {
      version.current = answer.value.version;
      setPost(answer.value);
      setRefusal("");
      setState("saved");
      return;
    }
    if (answer.refusal?.version !== undefined) {
      setRefusal(answer.error ?? "");
      setState("conflict");
      return;
    }
    setRefusal(answer.error ?? "");
    setState("refused");
  }, [draft, id]);

  useEffect(() => {
    if (state !== "dirty") return;
    const timer = setTimeout(() => void save(), AUTOSAVE_PAUSE);
    return () => clearTimeout(timer);
  }, [state, save]);

  function change(patch: Partial<Draft>) {
    setDraft((current) => (current ? { ...current, ...patch } : current));
    setState("dirty");
  }

  async function release() {
    setAsking(false);
    if (state === "dirty" || state === "refused") await save();
    const answer = await publishPost(id);
    if (answer.error || !answer.value) {
      setRefusal(answer.error ?? "");
      return;
    }
    setPost(answer.value);
    version.current = answer.value.version;
    setRefusal("");
    setState("clean");
  }

  if (account === undefined) {
    return <p className={styles.loading}>Checking your account…</p>;
  }
  if (!account) {
    return (
      <ConsoleGate
        action="Sign in"
        heading="Sign in to write"
        href="/sign-in"
        line="Illarin opens the editor to the accounts it has approved."
      />
    );
  }
  if (!post || !draft) {
    return (
      <p className={styles.loading} aria-live="polite">
        {failure || "Opening the post…"}
      </p>
    );
  }

  const category =
    categories.find((one) => one.id === draft.categoryId) ?? post.category;

  return (
    <div className={styles.writer}>
      <h1 className={styles.heading}>{draft.title || "Untitled post"}</h1>
      <header className={styles.bar}>
        <Link className={styles.back} href="/admin/blog">
          <ArrowLeft size={16} strokeWidth={1.8} aria-hidden="true" />
          Posts
        </Link>
        <p aria-live="polite" className={styles.state} data-state={state}>
          {stateWords(state, post)}
        </p>
        <div className={styles.actions}>
          <button
            className={styles.preview}
            onClick={() => setPreviewing((open) => !open)}
            type="button"
          >
            {previewing ? (
              <PenLine size={15} strokeWidth={1.8} aria-hidden="true" />
            ) : (
              <Eye size={15} strokeWidth={1.8} aria-hidden="true" />
            )}
            {previewing ? "Write" : "Preview"}
          </button>
          <button
            className={styles.publish}
            disabled={state === "conflict"}
            onClick={() => setAsking(true)}
            type="button"
          >
            {post.status === "published" ? "Publish changes" : "Publish"}
          </button>
        </div>
      </header>

      {refusal ? (
        <p className={styles.refusal} role="alert">
          {refusal}
          {state === "conflict" ? (
            <button onClick={() => void load()} type="button">
              Open the newer copy
            </button>
          ) : null}
        </p>
      ) : null}

      <FormDialog
        acknowledge={false}
        commit={post.status === "published" ? "Publish changes" : "Publish"}
        hint={
          post.status === "published"
            ? "Readers see this edition from the moment you publish it."
            : "Publishing keeps this edition, fixes the address and records who wrote it. None of the three can be taken back."
        }
        onClose={() => setAsking(false)}
        onCommit={() => void release()}
        open={asking}
        title={
          post.status === "published"
            ? "Publish the changes?"
            : "Publish this post?"
        }
      >
        <p className={styles.confirm}>
          {draft.title}
          <span>illarin.xyz/blog/{draft.slug}</span>
        </p>
      </FormDialog>

      {previewing ? (
        <article className={styles.reading}>
          <ArticleIdentity
            byline={null}
            category={category.label}
            publishedAt={post.publishedAt ?? null}
            release={post.release ?? null}
            standing="A preview of the working copy. The byline is written when the post is first published."
            summary={draft.summary}
            title={draft.title}
            updatedAt={post.updatedPublicAt ?? null}
          />
          <div className={styles.prose}>
            <PostBody document={draft.document} />
          </div>
        </article>
      ) : (
        <div className={styles.desk}>
          <div className={styles.writing}>
            <label className={styles.titleLabel} htmlFor="post-title">
              Title
            </label>
            <GrowingText
              className={styles.title}
              id="post-title"
              maxLength={160}
              onChange={(title) => change({ title })}
              value={draft.title}
            />
            <label className={styles.summaryLabel} htmlFor="post-summary">
              Summary
            </label>
            <GrowingText
              className={styles.summary}
              id="post-summary"
              maxLength={320}
              onChange={(summary) => change({ summary })}
              placeholder="One or two sentences a reader sees before the article."
              value={draft.summary}
            />
            <BodyEditor
              document={draft.document}
              key={edition}
              onChange={(document) => change({ document })}
            />
          </div>
          <PostDetails
            apps={releaseApps(apps, post)}
            categories={categories.length > 0 ? categories : [post.category]}
            draft={draft}
            locked={post.status === "published"}
            onChange={change}
            post={post}
          />
        </div>
      )}
    </div>
  );
}

function asDraft(post: Post): Draft {
  return {
    categoryId: post.category.id,
    title: post.title,
    summary: post.summary,
    slug: post.slug,
    document: asPostDocument(post.document),
    release: post.release
      ? {
          appId: post.release.app.id,
          version: post.release.version,
          address: post.release.address ?? "",
        }
      : null,
  };
}

// releaseApps keeps a post's existing project listed even after it is retired.
function releaseApps(open: PublicationApp[], post: Post): PublicationApp[] {
  const named = post.release?.app;
  if (!named || open.some((app) => app.id === named.id)) return open;
  return [named, ...open];
}

function stateWords(state: Saving, post: Post): string {
  switch (state) {
    case "saving":
      return "Saving…";
    case "saved":
      return "Saved";
    case "dirty":
      return "Unsaved";
    case "conflict":
      return "Someone else saved this post";
    case "refused":
      return "Not saved";
    default:
      return post.status === "published" ? "Published" : "Draft";
  }
}
