"use client";

import {
  ArrowDown,
  ArrowUp,
  ImageUp,
  Plus,
  ShieldCheck,
  Trash2,
  X,
} from "lucide-react";
import Link from "next/link";
import {
  type FormEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { CreatorMark } from "@/components/media/CreatorMark";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { Profile, ProfileLink } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { ProfilePreview } from "./ProfilePreview";
import styles from "./PublicProfileEditor.module.css";

const BIOGRAPHY_LIMIT = 400;
const LINK_LIMIT = 6;
const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

type Draft = {
  displayName: string;
  biography: string;
  contactEmail: string;
  links: ProfileLink[];
};

type Answer = Profile & { error?: string; field?: string };

function FieldError({
  id,
  shown,
  children,
}: {
  id: string;
  shown: boolean;
  children: string;
}) {
  if (!shown || !children) return null;
  return (
    <p className={styles.fieldError} id={id}>
      {children}
    </p>
  );
}

function draftOf(profile: Profile): Draft {
  return {
    displayName: profile.displayName,
    biography: profile.biography,
    contactEmail: profile.contactEmail,
    links: profile.links,
  };
}

export function PublicProfileEditor() {
  const { account } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [message, setMessage] = useState("");
  const [failedField, setFailedField] = useState("");
  const [fieldMessage, setFieldMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const [avatarPending, setAvatarPending] = useState(false);
  const [confirmingRemoval, setConfirmingRemoval] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const handle = account?.handle;

  useEffect(() => {
    if (!handle) return;
    let current = true;
    void (async () => {
      const response = await fetch(`/api/v1/profiles/${handle}`, {
        cache: "no-store",
        credentials: "same-origin",
      });
      if (!response.ok || !current) return;
      const found = (await response.json()) as Profile;
      if (!current) return;
      setProfile(found);
      setDraft(draftOf(found));
    })();
    return () => {
      current = false;
    };
  }, [handle]);

  const accept = useCallback((answer: Profile) => {
    setProfile(answer);
    setDraft(draftOf(answer));
  }, []);

  const acceptAvatar = useCallback((answer: Profile) => {
    setProfile(answer);
  }, []);

  if (account === undefined || (account && !draft)) {
    return (
      <p className={styles.loading} aria-live="polite">
        Reading your public profile…
      </p>
    );
  }

  if (!account) {
    return (
      <section className={styles.gate}>
        <ShieldCheck size={27} strokeWidth={1.35} aria-hidden="true" />
        <h2>Sign in to edit your public profile</h2>
        <p>
          Your public profile belongs to your account, so it needs a session.
        </p>
        <Link href="/sign-in">Sign in</Link>
      </section>
    );
  }

  if (!account.emailVerified) {
    return (
      <section className={styles.gate}>
        <ShieldCheck size={27} strokeWidth={1.35} aria-hidden="true" />
        <h2>Verify your email to publish a profile</h2>
        <p>
          Your handle stays public either way. The rest of the profile opens
          once an address is verified.
        </p>
        <Link href="/verify-email">Verify email</Link>
      </section>
    );
  }

  if (!draft || !profile) return null;

  function change(field: keyof Draft, value: string) {
    setDraft((current) => (current ? { ...current, [field]: value } : current));
  }

  function changeLink(index: number, field: keyof ProfileLink, value: string) {
    setDraft((current) =>
      current
        ? {
            ...current,
            links: current.links.map((link, position) =>
              position === index ? { ...link, [field]: value } : link,
            ),
          }
        : current,
    );
  }

  function addLink() {
    setDraft((current) =>
      current
        ? { ...current, links: [...current.links, { label: "", address: "" }] }
        : current,
    );
  }

  function removeLink(index: number) {
    setDraft((current) =>
      current
        ? { ...current, links: current.links.filter((_, at) => at !== index) }
        : current,
    );
  }

  function moveLink(index: number, step: number) {
    setDraft((current) => {
      if (!current) return current;
      const destination = index + step;
      if (destination < 0 || destination >= current.links.length)
        return current;
      const links = [...current.links];
      [links[index], links[destination]] = [links[destination], links[index]];
      return { ...current, links };
    });
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!draft) return;
    setSaving(true);
    setMessage("");
    setFailedField("");
    setFieldMessage("");
    try {
      const response = await browserFetch("/api/v1/account/profile", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(draft),
      });
      const answer = (await response.json()) as Answer;
      if (!response.ok) {
        const reason = answer.error ?? "The profile could not be saved.";
        setFailedField(answer.field ?? "");
        setFieldMessage(answer.field ? reason : "");
        setMessage(
          answer.field ? "Check the marked field and save again." : reason,
        );
        return;
      }
      accept(answer);
      setMessage("Your public profile is saved.");
    } catch {
      setMessage(UNREACHABLE);
    } finally {
      setSaving(false);
    }
  }

  async function uploadAvatar(file: File) {
    setAvatarPending(true);
    setMessage("");
    const body = new FormData();
    body.append("file", file);
    try {
      const response = await browserFetch("/api/v1/account/profile/avatar", {
        method: "PUT",
        credentials: "same-origin",
        body,
      });
      const answer = (await response.json()) as Answer;
      if (!response.ok) {
        setMessage(answer.error ?? "That image could not be used.");
        return;
      }
      acceptAvatar(answer);
      setMessage("Your avatar is updated.");
    } catch {
      setMessage(UNREACHABLE);
    } finally {
      setAvatarPending(false);
      if (fileInput.current) fileInput.current.value = "";
    }
  }

  async function removeAvatar() {
    setAvatarPending(true);
    setConfirmingRemoval(false);
    setMessage("");
    try {
      const response = await browserFetch("/api/v1/account/profile/avatar", {
        method: "DELETE",
        credentials: "same-origin",
      });
      const answer = (await response.json()) as Answer;
      if (!response.ok) {
        setMessage(answer.error ?? "The avatar could not be removed.");
        return;
      }
      acceptAvatar(answer);
      setMessage("Your avatar is removed.");
    } catch {
      setMessage(UNREACHABLE);
    } finally {
      setAvatarPending(false);
    }
  }

  const remaining = BIOGRAPHY_LIMIT - draft.biography.length;

  return (
    <div className={styles.desk}>
      <form className={styles.editor} onSubmit={save} noValidate>
        <div className={styles.portrait}>
          <CreatorMark handle={profile.handle} portrait={profile.avatar} />
          <div className={styles.portraitCopy}>
            <h2>Avatar</h2>
            <p>
              A square PNG, JPEG, WebP or GIF reads best. Illarin serves it back
              at one size.
            </p>
            <div className={styles.portraitActions}>
              <button
                type="button"
                className={styles.secondaryAction}
                onClick={() => fileInput.current?.click()}
                disabled={avatarPending}
              >
                <ImageUp size={15} strokeWidth={1.6} aria-hidden="true" />
                {avatarPending
                  ? "Working…"
                  : profile.avatar
                    ? "Replace image"
                    : "Upload image"}
              </button>
              {profile.avatar && !confirmingRemoval ? (
                <button
                  type="button"
                  className={styles.quietAction}
                  onClick={() => setConfirmingRemoval(true)}
                  disabled={avatarPending}
                >
                  <Trash2 size={15} strokeWidth={1.6} aria-hidden="true" />
                  Remove
                </button>
              ) : null}
              {profile.avatar && confirmingRemoval ? (
                <span className={styles.confirm}>
                  Remove it? Your mark takes its place.
                  <button
                    type="button"
                    className={styles.criticalAction}
                    onClick={removeAvatar}
                    disabled={avatarPending}
                  >
                    Remove
                  </button>
                  <button
                    type="button"
                    className={styles.quietAction}
                    onClick={() => setConfirmingRemoval(false)}
                  >
                    Keep
                  </button>
                </span>
              ) : null}
            </div>
            <input
              ref={fileInput}
              className="sr-only"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) void uploadAvatar(file);
              }}
            />
          </div>
        </div>

        <div className={styles.field}>
          <label htmlFor="profile-display-name">Display name</label>
          <input
            id="profile-display-name"
            name="displayName"
            type="text"
            maxLength={48}
            autoComplete="nickname"
            placeholder={`@${profile.handle}`}
            value={draft.displayName}
            aria-invalid={failedField === "displayName" || undefined}
            aria-describedby={
              failedField === "displayName"
                ? "profile-display-name-error"
                : undefined
            }
            onChange={(event) => change("displayName", event.target.value)}
          />
          <FieldError
            id="profile-display-name-error"
            shown={failedField === "displayName"}
          >
            {fieldMessage}
          </FieldError>
          <p>
            Shown above your handle. It does not have to be unique, and clearing
            it puts your handle back in its place.
          </p>
        </div>

        <div className={styles.field}>
          <label htmlFor="profile-biography">Biography</label>
          <textarea
            id="profile-biography"
            name="biography"
            rows={4}
            maxLength={BIOGRAPHY_LIMIT}
            value={draft.biography}
            aria-invalid={failedField === "biography" || undefined}
            aria-describedby={
              failedField === "biography"
                ? "profile-biography-error"
                : undefined
            }
            onChange={(event) => change("biography", event.target.value)}
          />
          <FieldError
            id="profile-biography-error"
            shown={failedField === "biography"}
          >
            {fieldMessage}
          </FieldError>
          <p className={styles.hint}>
            <span>Plain text, no formatting.</span>
            <span
              className={styles.count}
              data-low={remaining <= 40 || undefined}
            >
              {remaining} left
            </span>
          </p>
        </div>

        <div className={styles.field}>
          <label htmlFor="profile-contact">Public contact email</label>
          <input
            id="profile-contact"
            name="contactEmail"
            type="email"
            maxLength={254}
            autoComplete="off"
            placeholder="Leave empty to show no address"
            value={draft.contactEmail}
            aria-invalid={failedField === "contactEmail" || undefined}
            aria-describedby={
              failedField === "contactEmail"
                ? "profile-contact-error"
                : undefined
            }
            onChange={(event) => change("contactEmail", event.target.value)}
          />
          <FieldError
            id="profile-contact-error"
            shown={failedField === "contactEmail"}
          >
            {fieldMessage}
          </FieldError>
          <p>
            Anyone can read this. It is separate from your sign-in address,
            which stays private and is never copied here.
          </p>
        </div>

        <fieldset
          className={styles.links}
          data-invalid={failedField === "links" || undefined}
        >
          <legend>Links</legend>
          <p className={styles.linksHint}>
            Up to {LINK_LIMIT} addresses, in the order you list them. Each one
            needs a label and an https address.
          </p>
          <FieldError id="profile-links-error" shown={failedField === "links"}>
            {fieldMessage}
          </FieldError>
          {draft.links.length > 0 ? (
            <ol className={styles.linkList}>
              {draft.links.map((link, index) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: a link's position is its identity here
                <li key={index}>
                  <span className={styles.linkPosition}>{index + 1}</span>
                  <input
                    type="text"
                    maxLength={32}
                    placeholder="Label"
                    aria-label={`Link ${index + 1} label`}
                    value={link.label}
                    onChange={(event) =>
                      changeLink(index, "label", event.target.value)
                    }
                  />
                  <input
                    type="url"
                    maxLength={300}
                    placeholder="https://example.com"
                    aria-label={`Link ${index + 1} address`}
                    value={link.address}
                    onChange={(event) =>
                      changeLink(index, "address", event.target.value)
                    }
                  />
                  <div className={styles.linkOrder}>
                    <button
                      type="button"
                      onClick={() => moveLink(index, -1)}
                      disabled={index === 0}
                      aria-label={`Move link ${index + 1} up`}
                    >
                      <ArrowUp size={15} strokeWidth={1.7} aria-hidden="true" />
                    </button>
                    <button
                      type="button"
                      onClick={() => moveLink(index, 1)}
                      disabled={index === draft.links.length - 1}
                      aria-label={`Move link ${index + 1} down`}
                    >
                      <ArrowDown
                        size={15}
                        strokeWidth={1.7}
                        aria-hidden="true"
                      />
                    </button>
                    <button
                      type="button"
                      onClick={() => removeLink(index)}
                      aria-label={`Remove link ${index + 1}`}
                    >
                      <X size={15} strokeWidth={1.7} aria-hidden="true" />
                    </button>
                  </div>
                </li>
              ))}
            </ol>
          ) : null}
          {draft.links.length < LINK_LIMIT ? (
            <button
              type="button"
              className={styles.secondaryAction}
              onClick={addLink}
            >
              <Plus size={15} strokeWidth={1.7} aria-hidden="true" />
              Add link
            </button>
          ) : null}
        </fieldset>

        <div className={styles.commit}>
          <button
            type="submit"
            className={styles.primaryAction}
            disabled={saving}
          >
            {saving ? "Saving…" : "Save profile"}
          </button>
          <Link className={styles.view} href={`/@${profile.handle}`}>
            View public profile
          </Link>
          {message ? (
            <output className={styles.message} aria-live="polite">
              {message}
            </output>
          ) : null}
        </div>
      </form>
      <ProfilePreview
        handle={profile.handle}
        avatar={profile.avatar}
        displayName={draft.displayName}
        biography={draft.biography}
        contactEmail={draft.contactEmail}
        links={draft.links}
      />
    </div>
  );
}
