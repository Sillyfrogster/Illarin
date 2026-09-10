"use client";

import Image from "next/image";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, Said, TextInput, Trouble } from "@/components/ui/field";
import { correctPostAddress, correctPostByline } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { normalizedSlug } from "@/lib/post-link";

type Part = "address" | "name";

/** What publishing fixed: the address readers have and the name on the post. */
export function PublishedIdentity({
  admin,
  post,
  onCorrected,
}: {
  admin: boolean;
  post: Post;
  onCorrected: (post: Post) => void;
}) {
  const [changing, setChanging] = useState<Part | null>(null);
  const [failure, setFailure] = useState("");
  const [changed, setChanged] = useState("");

  function done(corrected: Post, said: string) {
    setChanging(null);
    setFailure("");
    setChanged(said);
    onCorrected(corrected);
  }

  function open(part: Part) {
    setChanging(part);
    setFailure("");
    setChanged("");
  }

  return (
    <section
      aria-labelledby="published-identity"
      className="flex flex-col gap-5 rounded-plate bg-deep p-5"
    >
      <div>
        <h3
          className="font-display text-ui font-medium text-ink"
          id="published-identity"
        >
          Fixed when you published
        </h3>
        <p className="mt-1 font-prose text-meta text-mute">
          {admin
            ? "Readers already have these. Change one only to fix a mistake."
            : "Readers already have these. Ask an admin if either one is wrong."}
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <p className="font-ui text-label font-medium text-mute">Address</p>
        <p className="font-prose text-meta text-ink wrap-anywhere">
          <span className="text-mute">illarin.xyz/blog/</span>
          {post.slug}
        </p>
        {post.formerAddresses.length > 0 ? (
          <div className="font-prose text-meta text-mute">
            <p>Still works too:</p>
            <ul className="mt-1 list-none">
              {post.formerAddresses.map((address) => (
                <li className="wrap-anywhere" key={address}>
                  <span className="opacity-70">illarin.xyz/blog/</span>
                  {address}
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        {changing === "address" ? (
          <ChangeAddress
            onCancel={() => setChanging(null)}
            onDone={done}
            onFailure={setFailure}
            post={post}
          />
        ) : admin ? (
          <Button
            className="self-start"
            onClick={() => open("address")}
            size="compact"
          >
            Change the address
          </Button>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <p className="font-ui text-label font-medium text-mute">
          Name on the post
        </p>
        {post.byline ? <Named post={post} /> : null}
        {changing === "name" ? (
          <ChangeName
            onCancel={() => setChanging(null)}
            onDone={done}
            onFailure={setFailure}
            post={post}
          />
        ) : admin ? (
          <Button
            className="self-start"
            onClick={() => open("name")}
            size="compact"
          >
            Change the name
          </Button>
        ) : null}
      </div>

      {failure ? <Trouble>{failure}</Trouble> : null}
      {changed ? <Said>{changed}</Said> : null}
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
    <p className="flex items-center gap-3">
      <span className="grid size-9 shrink-0 place-items-center overflow-hidden rounded-full bg-plane font-display text-meta font-medium text-mute">
        {byline.avatar ? (
          <Image
            alt=""
            className="size-9 object-cover"
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
      <span className="min-w-0">
        <span className="block font-ui text-ui text-ink wrap-anywhere">
          {name}
        </span>
        <span className="block font-prose text-meta text-mute wrap-anywhere">
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
    <form className="flex flex-col gap-3" onSubmit={commit}>
      <Field
        hint={`${post.slug} keeps working and goes to the new address. No other post can ever use it.`}
        htmlFor="corrected-address"
        label="New address"
      >
        <TextInput
          className="bg-plane"
          disabled={busy}
          id="corrected-address"
          maxLength={80}
          onChange={(event) => setSlug(event.target.value)}
          ref={field}
          value={slug}
        />
      </Field>
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
    <form className="flex flex-col gap-3" onSubmit={commit}>
      <Field
        hint="Their name, picture and jobs are copied onto the post again. Nothing else about the post changes."
        htmlFor="corrected-name"
        label="Handle of the person who wrote it"
      >
        <TextInput
          className="bg-plane"
          disabled={busy}
          id="corrected-name"
          maxLength={40}
          onChange={(event) => setHandle(event.target.value)}
          ref={field}
          value={handle}
        />
      </Field>
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
    <div className="flex flex-wrap items-center gap-2">
      <Button
        disabled={!ready}
        loading={busy}
        size="compact"
        type="submit"
        variant="primary"
      >
        {busy ? working : commit}
      </Button>
      <Button disabled={busy} onClick={onCancel} size="compact" variant="ghost">
        Cancel
      </Button>
    </div>
  );
}
