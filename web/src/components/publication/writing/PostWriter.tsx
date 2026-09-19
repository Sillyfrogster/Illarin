"use client";

import { AnimatePresence } from "framer-motion";
import {
  ArrowLeft,
  BookOpen,
  History,
  PencilLine,
  SlidersHorizontal,
} from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { Gate } from "@/components/ui/gate";
import { PageWaiting } from "@/components/ui/waiting";
import {
  Dock,
  DockAction,
  type DockState,
  DockTool,
} from "@/components/workspace/Dock";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import {
  keepPostVersion,
  readPost,
  saveDraftedChanges,
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
import { readableDate } from "@/lib/dates";
import {
  type Draft,
  draftFromPost,
  mergedMedia,
  namedRelease,
  publicationLabel,
  type Saving,
  savingWords,
  writerStanding,
} from "@/lib/post-writing";
import { ArticleIdentity } from "../ArticleIdentity";
import { PostBody } from "../PostBody";
import { DetailsRail } from "./DetailsRail";
import { GrowingText } from "./GrowingText";
import { HistoryRail } from "./HistoryRail";
import { PostNotices } from "./PostNotices";
import { PublicationRail } from "./PublicationRail";
import { WritingSurface } from "./WritingSurface";

const AUTOSAVE_PAUSE = 1200;

type Rail = "details" | "history" | "publication" | null;

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
  const [reading, setReading] = useState(false);
  const [rail, setRail] = useState<Rail>(null);
  const [keeping, setKeeping] = useState(false);
  const [said, setSaid] = useState("");
  const [edition, setEdition] = useState(0);
  const [stamp, setStamp] = useState(0);
  const version = useRef(0);
  const writing = useRef<HTMLElement | null>(null);
  const place = useRef(0);

  const settle = useCallback((written: Post, rewrite: boolean) => {
    setPost(written);
    version.current = written.version;
    setRefusal("");
    setState("clean");
    if (!rewrite) return;
    setDraft(draftFromPost(written));
    setMedia(written.media);
    setEdition((count) => count + 1);
  }, []);

  const load = useCallback(async () => {
    const answer = await readPost(id);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    settle(answer.value, true);
  }, [id, settle]);

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
    if (!draft) return version.current;
    setState("saving");
    const answer = await saveDraftedChanges(id, {
      version: version.current,
      ...draft,
    });
    const written = answer.value;
    if (written) {
      version.current = written.version;
      setPost(written);
      setMedia((held) => mergedMedia(held, written.media));
      setRefusal("");
      setState("saved");
      return written.version;
    }
    setRefusal(answer.error ?? "");
    setState(answer.refusal?.version !== undefined ? "conflict" : "refused");
    return version.current;
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
        setRefusal(answer.error ?? "That picture could not be uploaded.");
        return null;
      }
      setMedia((held) => [...held, added]);
      return added;
    },
    [id],
  );

  function change(patch: Partial<Draft>) {
    setDraft((current) => (current ? { ...current, ...patch } : current));
    setState("dirty");
  }

  function look(atReading: boolean) {
    if (atReading && !reading) place.current = window.scrollY;
    setReading(atReading);
    if (!atReading && reading) {
      requestAnimationFrame(() => {
        window.scrollTo({ top: place.current });
        writing.current?.focus({ preventScroll: true });
      });
    }
  }

  async function keep() {
    setKeeping(true);
    const at =
      state === "dirty" || state === "refused" ? await save() : version.current;
    const answer = await keepPostVersion(id, at);
    setKeeping(false);
    if (answer.error || !answer.value) {
      setRefusal(
        answer.error ?? "The checkpoint could not be saved. Try again.",
      );
      return;
    }
    setRefusal("");
    setSaid(`Saved as revision ${answer.value.number}.`);
    setStamp((count) => count + 1);
  }

  if (account === undefined) {
    return <PageWaiting>Checking your account…</PageWaiting>;
  }

  if (!account) {
    return (
      <div className={`${shellClasses} pt-14`}>
        <Gate
          action="Sign in"
          heading="Sign in to write"
          href="/sign-in"
          line="Sign in with an account that has publishing access."
        />
      </div>
    );
  }

  if (!post || !draft) {
    return <PageWaiting>{failure || "Opening the post…"}</PageWaiting>;
  }

  const category =
    categories.find((one) => one.id === draft.categoryId) ?? post.category;
  const deleted = writerStanding(post) === "deleted";
  const showingReader = reading || deleted;

  return (
    <div className={`${shellClasses} pt-6 pb-40`}>
      <div className="mx-auto max-w-[64rem]">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link
            className="inline-flex min-h-11 items-center gap-2 font-ui text-meta text-mute outline-offset-3 hover:text-ink"
            href="/admin/blog"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            Your posts
          </Link>
          {said ? (
            <output
              aria-live="polite"
              className="font-prose text-meta text-accent"
            >
              {said}
            </output>
          ) : null}
        </div>

        <PostNotices
          onOpenPublication={() => setRail("publication")}
          post={post}
        />

        {showingReader ? (
          <article className="mt-group">
            <ArticleIdentity
              byline={post.byline ?? null}
              category={category.label}
              header={draft.header}
              media={media}
              publishedAt={post.publishedAt ?? null}
              release={post.release ?? null}
              standing={
                deleted
                  ? "This post is deleted. Restore it to continue editing."
                  : "Preview. Your byline is added at first publication."
              }
              summary={draft.summary}
              title={draft.title}
              updatedAt={post.updatedPublicAt ?? null}
            />
            <div className="mt-group max-w-[70ch]">
              <PostBody document={draft.document} media={media} />
            </div>
          </article>
        ) : null}

        {deleted ? null : (
          <div
            className="mt-group"
            hidden={showingReader}
            onFocusCapture={(event) => {
              const held = writingUnder(event.target);
              if (held) writing.current = held;
            }}
          >
            <label className="sr-only" htmlFor="post-title">
              Title
            </label>
            <GrowingText
              className="w-full min-h-11 resize-none border-0 bg-transparent p-0 font-display text-[clamp(2.1rem,4.3vw,3.4rem)]/[1.08] font-medium tracking-[-0.03em] break-words text-ink outline-offset-4 placeholder:text-mute/60"
              id="post-title"
              maxLength={160}
              onChange={(title) => change({ title })}
              placeholder="Title"
              shown={!showingReader}
              value={draft.title}
            />
            <label className="sr-only" htmlFor="post-summary">
              Summary
            </label>
            <GrowingText
              className="mt-6 min-h-11 w-full max-w-[54ch] resize-none border-0 bg-transparent p-0 font-prose text-lede text-mute outline-offset-4 placeholder:text-mute/60"
              id="post-summary"
              maxLength={320}
              onChange={(summary) => change({ summary })}
              placeholder="One or two sentences a reader sees before the article."
              shown={!showingReader}
              value={draft.summary}
            />
            <div className="max-w-[70ch]">
              <WritingSurface
                document={draft.document}
                key={edition}
                media={media}
                onChange={(document) => change({ document })}
                onUpload={(file) => upload("document", file)}
              />
            </div>
          </div>
        )}
      </div>

      <Dock
        actions={
          <>
            {deleted ? null : (
              <DockAction
                disabled={
                  state === "saving" || state === "clean" || state === "saved"
                }
                onClick={() => void save()}
              >
                {state === "saving" ? "Saving…" : "Save"}
              </DockAction>
            )}
            <DockAction
              onClick={() =>
                setRail((open) =>
                  open === "publication" ? null : "publication",
                )
              }
              strong
            >
              {publicationLabel(post)}
            </DockAction>
          </>
        }
        detail={detail(state, post, refusal)}
        railOpen={rail !== null}
        state={lightOf(state, post)}
        tools={
          <>
            {deleted ? null : (
              <DockTool
                active={reading}
                icon={reading ? PencilLine : BookOpen}
                label={reading ? "Continue editing" : "Preview"}
                onClick={() => look(!reading)}
              />
            )}
            {deleted ? null : (
              <DockTool
                active={rail === "details"}
                icon={SlidersHorizontal}
                label="Details"
                onClick={() =>
                  setRail((open) => (open === "details" ? null : "details"))
                }
              />
            )}
            <DockTool
              active={rail === "history"}
              icon={History}
              label="Editorial history"
              onClick={() =>
                setRail((open) => (open === "history" ? null : "history"))
              }
            />
          </>
        }
        words={savingWords(state, post)}
      />

      <AnimatePresence>
        {state === "conflict" ? (
          <WorkspaceRail
            description={refusal || "This post was saved somewhere else."}
            key="conflict"
            title="A newer copy of this post exists"
            tone="stop"
          >
            <div className="flex flex-col gap-5">
              <p className="font-prose text-ui text-mute">
                Your writing is still on the page. Copy anything worth keeping,
                then open the newer copy to work from it.
              </p>
              <button
                className="inline-flex min-h-11 items-center justify-center rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3"
                onClick={() => void load()}
                type="button"
              >
                Open the newer copy
              </button>
            </div>
          </WorkspaceRail>
        ) : null}

        {rail === "details" && state !== "conflict" ? (
          <WorkspaceRail
            description="Everything about the post that is not the writing itself."
            key="details"
            onClose={() => setRail(null)}
            title="Details"
          >
            <DetailsRail
              admin={admin}
              apps={namedRelease(apps, post)}
              categories={categories.length > 0 ? categories : [post.category]}
              draft={draft}
              locked={post.publishedAt !== undefined}
              media={media}
              onChange={change}
              onCorrected={(corrected) => {
                settle(corrected, false);
                setDraft((current) =>
                  current ? { ...current, slug: corrected.slug } : current,
                );
              }}
              onUpload={upload}
              post={post}
            />
          </WorkspaceRail>
        ) : null}

        {rail === "history" && state !== "conflict" ? (
          <WorkspaceRail
            description="Save checkpoints and restore earlier revisions to your drafted changes."
            key="history"
            onClose={() => setRail(null)}
            title="Editorial history"
          >
            <HistoryRail
              handle={account.handle}
              keeping={keeping}
              onFailure={setRefusal}
              onKeep={() => void keep()}
              onRestored={(restored) => {
                settle(restored, true);
                setSaid("");
              }}
              post={post}
            />
          </WorkspaceRail>
        ) : null}

        {rail === "publication" && state !== "conflict" ? (
          <WorkspaceRail
            description="Manage publication, scheduling, withdrawal and recovery."
            key="publication"
            onClose={() => setRail(null)}
            title="Publication"
          >
            <PublicationRail
              admin={admin}
              onFailure={setRefusal}
              onSaveFirst={async () =>
                state === "dirty" || state === "refused"
                  ? await save()
                  : version.current
              }
              onSettled={(settled) => {
                settle(settled, false);
                setSaid("");
                setStamp((count) => count + 1);
              }}
              post={post}
              stamp={stamp}
            />
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function lightOf(state: Saving, post: Post): DockState {
  if (state === "conflict" || state === "refused") return "failed";
  if (state === "dirty") return "unsaved";
  if (state === "saving") return "saving";
  return writerStanding(post) === "published" ? "published" : "private";
}

function detail(state: Saving, post: Post, refusal: string): string {
  if (state === "conflict") return "Open the newer copy to carry on.";
  if (state === "refused")
    return refusal || "Your writing is still on this page. Try saving again.";
  const standing = writerStanding(post);
  if (standing === "deleted") return "Restore the post to continue editing.";
  if (standing === "withdrawn")
    return "The post is withdrawn from public view.";
  if (standing !== "published") return "Only you can open this post.";
  if (state === "dirty" || state === "saved") {
    return "Readers do not have your changes yet.";
  }
  const at = post.updatedPublicAt ?? post.publishedAt;
  return at ? `Published ${readableDate(at)}.` : "All changes are published.";
}

function writingUnder(target: EventTarget): HTMLElement | null {
  if (!(target instanceof HTMLElement)) return null;
  return target.closest("textarea, .ProseMirror");
}
