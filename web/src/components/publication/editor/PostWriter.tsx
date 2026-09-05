"use client";

import { ArrowLeft, BookmarkPlus } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { ConsoleGate } from "@/components/console/ConsoleGate";
import { FormDialog } from "@/components/console/FormDialog";
import { ArticleHeader } from "@/components/publication/ArticleHeader";
import { ArticleIdentity } from "@/components/publication/ArticleIdentity";
import { PostBody } from "@/components/publication/PostBody";
import {
  keepPostVersion,
  publishPost,
  readPost,
  saveWorkingCopy,
  schedulePost,
  uploadPostMedia,
} from "@/lib/api/posts";
import { readWorkspace } from "@/lib/api/publication";
import type {
  Post,
  PostMedia,
  PostMediaPurpose,
  PublicationApp,
  PublicationCategory,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { asPostDocument, type PostDocument } from "@/lib/post-document";
import {
  atLeastAnHourAhead,
  type LocalParts,
  toInstant,
} from "@/lib/schedule-time";
import { BodyEditor } from "./BodyEditor";
import { GrowingText } from "./GrowingText";
import { PostDetails } from "./PostDetails";
import { PostHistory } from "./PostHistory";
import styles from "./PostWriter.module.css";
import { ScheduleBand } from "./ScheduleBand";
import { ScheduleFields } from "./ScheduleFields";

/** How long the writer pauses before the working copy is saved. */
const AUTOSAVE_PAUSE = 1200;

type Draft = {
  categoryId: string;
  title: string;
  summary: string;
  slug: string;
  document: PostDocument;
  release: { appId: string; version: string; address: string } | null;
  header: { mediaId: string; alt: string; caption: string } | null;
  socialMediaId: string | null;
};

type Saving = "clean" | "dirty" | "saving" | "saved" | "conflict" | "refused";

type Door = "now" | "later";

type View = "write" | "preview" | "history";

const VIEWS: { view: View; label: string }[] = [
  { view: "write", label: "Write" },
  { view: "preview", label: "Preview" },
  { view: "history", label: "History" },
];

export function PostWriter({ id }: { id: string }) {
  const { account } = useAuth();
  const [post, setPost] = useState<Post | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [media, setMedia] = useState<PostMedia[]>([]);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [apps, setApps] = useState<PublicationApp[]>([]);
  const [admin, setAdmin] = useState(false);
  const [state, setState] = useState<Saving>("clean");
  const [refusal, setRefusal] = useState("");
  const [failure, setFailure] = useState("");
  const [view, setView] = useState<View>("write");
  const [asking, setAsking] = useState(false);
  const [door, setDoor] = useState<Door>("now");
  const [when, setWhen] = useState<LocalParts>({ date: "", time: "" });
  const [keeping, setKeeping] = useState(false);
  const [kept, setKept] = useState("");
  const [edition, setEdition] = useState(0);
  const [stamp, setStamp] = useState(0);
  const version = useRef(0);
  const writing = useRef<HTMLElement | null>(null);

  const load = useCallback(async () => {
    const answer = await readPost(id);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setPost(answer.value);
    setDraft(asDraft(answer.value));
    setMedia(answer.value.media);
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
      setAdmin(answer.value.admin);
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
    const written = answer.value;
    if (written) {
      version.current = written.version;
      setPost(written);
      setMedia((held) => refreshed(held, written.media));
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

  const upload = useCallback(
    async (purpose: PostMediaPurpose, file: File) => {
      const answer = await uploadPostMedia(id, purpose, file);
      const added = answer.value;
      if (!added) {
        setRefusal(answer.error ?? "");
        return null;
      }
      setMedia((held) => [...held, added]);
      return added;
    },
    [id],
  );

  function look(at: View) {
    if (view === "write") writing.current = focused();
    setView(at);
    if (at === "write") queueMicrotask(() => writing.current?.focus());
  }

  function change(patch: Partial<Draft>) {
    setDraft((current) => (current ? { ...current, ...patch } : current));
    setState("dirty");
  }

  function corrected(post: Post) {
    setPost(post);
    version.current = post.version;
    setDraft((current) =>
      current ? { ...current, slug: post.slug } : current,
    );
  }

  function restored(post: Post) {
    setPost(post);
    setDraft(asDraft(post));
    setMedia(post.media);
    version.current = post.version;
    setState("clean");
    setRefusal("");
    setKept("");
    setEdition((count) => count + 1);
  }

  function askAt(door: Door) {
    setDoor(door);
    setWhen(atLeastAnHourAhead());
    setAsking(true);
  }

  async function release() {
    setAsking(false);
    if (state === "dirty" || state === "refused") await save();
    const answer =
      door === "later"
        ? await schedulePost(
            id,
            version.current,
            toInstant(when.date, when.time),
          )
        : await publishPost(id, version.current);
    if (answer.error || !answer.value) {
      setRefusal(answer.error ?? "");
      return;
    }
    settled(answer.value);
  }

  function settled(post: Post) {
    setPost(post);
    version.current = post.version;
    setRefusal("");
    setKept("");
    setState("clean");
    setStamp((count) => count + 1);
  }

  async function keep() {
    setKeeping(true);
    if (state === "dirty" || state === "refused") await save();
    const answer = await keepPostVersion(id, version.current);
    setKeeping(false);
    if (answer.error || !answer.value) {
      setRefusal(answer.error ?? "");
      return;
    }
    setRefusal("");
    setKept(`Kept as edition ${answer.value.number}.`);
    setStamp((count) => count + 1);
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
        <fieldset className={styles.views}>
          <legend className={styles.heading}>What you are looking at</legend>
          {VIEWS.map((one) => (
            <button
              aria-pressed={view === one.view}
              className={styles.view}
              key={one.view}
              onClick={() => look(one.view)}
              type="button"
            >
              {one.label}
            </button>
          ))}
        </fieldset>
        <div className={styles.actions}>
          <button
            className={styles.keep}
            disabled={keeping || state === "conflict"}
            onClick={() => void keep()}
            type="button"
          >
            <BookmarkPlus size={15} strokeWidth={1.8} aria-hidden="true" />
            {keeping ? "Keeping…" : "Keep this version"}
          </button>
          <button
            className={styles.publish}
            disabled={state === "conflict"}
            onClick={() => askAt("now")}
            type="button"
          >
            {post.status === "published" ? "Publish changes" : "Publish"}
          </button>
        </div>
      </header>

      <p aria-live="polite" className={styles.kept}>
        {kept}
      </p>

      <ScheduleBand
        onChanged={settled}
        onFailure={setRefusal}
        onScheduleAgain={() => askAt("later")}
        post={post}
      />

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
        commit={commitWords(door, post)}
        hint={publishHint(door, post)}
        onClose={() => setAsking(false)}
        onCommit={() => void release()}
        open={asking}
        ready={door === "now" || toInstant(when.date, when.time) !== ""}
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
        <fieldset className={styles.doors}>
          <legend>When</legend>
          <label className={styles.door}>
            <input
              checked={door === "now"}
              name="publish-door"
              onChange={() => setDoor("now")}
              type="radio"
            />
            <span>
              Now
              <span>Readers see it as soon as you press the button.</span>
            </span>
          </label>
          <label className={styles.door}>
            <input
              checked={door === "later"}
              name="publish-door"
              onChange={() => setDoor("later")}
              type="radio"
            />
            <span>
              At a set time
              <span>Illarin publishes this exact version for you.</span>
            </span>
          </label>
        </fieldset>
        {door === "later" ? (
          <ScheduleFields
            id="publish-schedule"
            onChange={setWhen}
            parts={when}
          />
        ) : null}
      </FormDialog>

      {view === "preview" ? (
        <article className={styles.reading}>
          <ArticleIdentity
            byline={null}
            category={category.label}
            publishedAt={post.publishedAt ?? null}
            release={post.release ?? null}
            standing="A preview. The name goes on when you first publish."
            summary={draft.summary}
            title={draft.title}
            updatedAt={post.updatedPublicAt ?? null}
          />
          <div className={styles.prose}>
            <ArticleHeader header={draft.header} media={media} />
            <PostBody document={draft.document} media={media} />
          </div>
        </article>
      ) : null}

      {view === "history" ? (
        <PostHistory
          handle={account.handle}
          key={stamp}
          onFailure={setRefusal}
          onRestored={restored}
          post={post}
        />
      ) : null}

      <div className={styles.desk} hidden={view !== "write"}>
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
            media={media}
            onChange={(document) => change({ document })}
            onUpload={(file) => upload("document", file)}
          />
        </div>
        <PostDetails
          admin={admin}
          apps={releaseApps(apps, post)}
          categories={categories.length > 0 ? categories : [post.category]}
          draft={draft}
          locked={post.status === "published"}
          media={media}
          onChange={change}
          onCorrected={corrected}
          onUpload={upload}
          post={post}
        />
      </div>
    </div>
  );
}

// publishHint says what this button will do, including to a waiting schedule.
function publishHint(door: Door, post: Post): string {
  const schedule = post.schedule;
  const waiting =
    schedule?.state === "pending" || schedule?.state === "publishing";
  if (door === "now" && waiting) {
    return "Publishing now stops the edition waiting to go live.";
  }
  if (post.status === "published") {
    return "The version in front of you is captured either way, and later edits do not change it.";
  }
  return "This fixes the address and puts your name on the post. Only an admin can change either afterwards.";
}

// commitWords names the button by the door the author chose.
function commitWords(door: Door, post: Post): string {
  if (door === "later") return "Schedule";
  return post.status === "published" ? "Publish changes" : "Publish";
}

function asDraft(post: Post): Draft {
  return {
    categoryId: post.category.id,
    title: post.title,
    summary: post.summary,
    slug: post.slug,
    document: asPostDocument(post.document),
    header: post.header
      ? {
          mediaId: post.header.mediaId,
          alt: post.header.alt,
          caption: post.header.caption ?? "",
        }
      : null,
    socialMediaId: post.socialMediaId ?? null,
    release: post.release
      ? {
          appId: post.release.app.id,
          version: post.release.version,
          address: post.release.address ?? "",
        }
      : null,
  };
}

// focused answers the writing element the keyboard is in, if it is in one.
function focused(): HTMLElement | null {
  const element = document.activeElement;
  if (!(element instanceof HTMLElement)) return null;
  return element.closest("textarea, .ProseMirror") as HTMLElement | null;
}

// refreshed keeps an upload the working copy has not been saved with yet.
function refreshed(held: PostMedia[], saved: PostMedia[]): PostMedia[] {
  const known = new Set(saved.map((one) => one.id));
  return [...saved, ...held.filter((one) => !known.has(one.id))];
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
