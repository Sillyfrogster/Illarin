"use client";

import Image from "next/image";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { correctPostAddress, correctPostByline } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { normalizedSlug } from "@/lib/post-link";
import styles from "./PublishedIdentity.module.css";

export function PublishedIdentity({
  admin,
  post,
  onCorrected,
}: {
  admin: boolean;
  post: Post;
  onCorrected: (post: Post) => void;
}) {
  const [changing, setChanging] = useState<"address" | "name" | null>(null);
  const [failure, setFailure] = useState("");
  const [changed, setChanged] = useState("");

  function done(corrected: Post, said: string) {
    setChanging(null);
    setFailure("");
    setChanged(said);
    onCorrected(corrected);
  }

  function open(part: "address" | "name") {
    setChanging(part);
    setFailure("");
    setChanged("");
  }

  return (
    <section aria-labelledby="published-identity" className={styles.panel}>
      <h3 className={styles.heading} id="published-identity">
        Fixed when you published
      </h3>
      <p className={styles.lead}>
        {admin
          ? "Readers already have these. Change one only to fix a mistake."
          : "Readers already have these. Ask an admin if either one is wrong."}
      </p>

      <div className={styles.part}>
        <p className={styles.what}>Address</p>
        <p className={styles.address}>
          <span className={styles.site}>illarin.xyz/blog/</span>
          {post.slug}
        </p>
        {post.formerAddresses.length > 0 ? (
          <div className={styles.former}>
            <p>Still works too:</p>
            <ul>
              {post.formerAddresses.map((address) => (
                <li key={address}>
                  <span className={styles.site}>illarin.xyz/blog/</span>
                  {address}
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        {changing === "address" ? (
          <ChangeAddress
            post={post}
            onCancel={() => setChanging(null)}
            onDone={done}
            onFailure={setFailure}
          />
        ) : admin ? (
          <button
            className={styles.open}
            onClick={() => open("address")}
            type="button"
          >
            Change the address
          </button>
        ) : null}
      </div>

      <div className={styles.part}>
        <p className={styles.what}>Name on the post</p>
        {post.byline ? <Named post={post} /> : null}
        {changing === "name" ? (
          <ChangeName
            post={post}
            onCancel={() => setChanging(null)}
            onDone={done}
            onFailure={setFailure}
          />
        ) : admin ? (
          <button
            className={styles.open}
            onClick={() => open("name")}
            type="button"
          >
            Change the name
          </button>
        ) : null}
      </div>

      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}
      <p aria-live="polite" className={styles.changed}>
        {changed}
      </p>
    </section>
  );
}

function Named({ post }: { post: Post }) {
  const byline = post.byline;
  if (!byline) return null;
  const name = byline.displayName || `@${byline.handle}`;
  const standing = byline.app
    ? byline.app.name
    : [...byline.positions, ...byline.distinctions].join(" · ");
  return (
    <p className={styles.named}>
      <span className={styles.portrait}>
        {byline.avatar ? (
          <Image
            alt=""
            height={34}
            src={byline.avatar.url}
            unoptimized
            width={34}
          />
        ) : (
          <span aria-hidden="true">
            {byline.handle.slice(0, 1).toUpperCase()}
          </span>
        )}
      </span>
      <span className={styles.who}>
        <span className={styles.name}>{name}</span>
        <span className={styles.standing}>
          {standing || `@${byline.handle}`}
        </span>
      </span>
    </p>
  );
}

// useOpenedField moves the keyboard to the field that replaced the button.
function useOpenedField() {
  const field = useRef<HTMLInputElement>(null);
  useEffect(() => {
    field.current?.focus();
    field.current?.select();
  }, []);
  return field;
}

function ChangeAddress({
  post,
  onCancel,
  onDone,
  onFailure,
}: {
  post: Post;
  onCancel: () => void;
  onDone: (post: Post, said: string) => void;
  onFailure: (message: string) => void;
}) {
  const [slug, setSlug] = useState(post.slug);
  const [busy, setBusy] = useState(false);
  const field = useOpenedField();
  const ready = slug.trim() !== "" && slug.trim() !== post.slug;

  async function commit(event: FormEvent) {
    event.preventDefault();
    if (!ready || busy) return;
    setBusy(true);
    const answer = await correctPostAddress(post.id, slug.trim());
    setBusy(false);
    if (!answer.value) {
      onFailure(answer.error ?? "The address did not change.");
      return;
    }
    onDone(
      answer.value,
      `The post is now at illarin.xyz/blog/${answer.value.slug}.`,
    );
  }

  return (
    <form className={styles.change} onSubmit={commit}>
      <label htmlFor="corrected-address">New address</label>
      <input
        disabled={busy}
        id="corrected-address"
        maxLength={80}
        onChange={(event) => setSlug(event.target.value)}
        ref={field}
        value={slug}
      />
      <p className={styles.warning}>
        {post.slug} keeps working and goes to the new address. No other post can
        ever use it.
      </p>
      <Commit
        busy={busy}
        commit={`Move it to /blog/${normalizedSlug(slug) || "…"}`}
        onCancel={onCancel}
        ready={ready}
        working="Moving…"
      />
    </form>
  );
}

function ChangeName({
  post,
  onCancel,
  onDone,
  onFailure,
}: {
  post: Post;
  onCancel: () => void;
  onDone: (post: Post, said: string) => void;
  onFailure: (message: string) => void;
}) {
  const [handle, setHandle] = useState(post.byline?.handle ?? "");
  const [busy, setBusy] = useState(false);
  const field = useOpenedField();
  const ready = handle.trim() !== "" && handle.trim() !== post.byline?.handle;

  async function commit(event: FormEvent) {
    event.preventDefault();
    if (!ready || busy) return;
    setBusy(true);
    const answer = await correctPostByline(post.id, handle.trim());
    setBusy(false);
    if (!answer.value) {
      onFailure(answer.error ?? "The name did not change.");
      return;
    }
    onDone(
      answer.value,
      `The post now carries @${answer.value.byline?.handle}.`,
    );
  }

  return (
    <form className={styles.change} onSubmit={commit}>
      <label htmlFor="corrected-name">Handle of the person who wrote it</label>
      <input
        disabled={busy}
        id="corrected-name"
        maxLength={40}
        onChange={(event) => setHandle(event.target.value)}
        ref={field}
        value={handle}
      />
      <p className={styles.warning}>
        Their name, picture and jobs are copied onto the post again. Nothing
        else about the post changes.
      </p>
      <Commit
        busy={busy}
        commit={`Put @${handle.trim() || "…"} on it`}
        onCancel={onCancel}
        ready={ready}
        working="Changing…"
      />
    </form>
  );
}

function Commit({
  busy,
  commit,
  ready,
  working,
  onCancel,
}: {
  busy: boolean;
  commit: string;
  ready: boolean;
  working: string;
  onCancel: () => void;
}) {
  return (
    <div className={styles.commit}>
      <button className={styles.apply} disabled={busy || !ready} type="submit">
        {busy ? working : commit}
      </button>
      <button
        className={styles.cancel}
        disabled={busy}
        onClick={onCancel}
        type="button"
      >
        Cancel
      </button>
    </div>
  );
}
