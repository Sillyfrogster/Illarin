"use client";

import { ShieldCheck, ShieldOff } from "lucide-react";
import Link from "next/link";
import {
  type FormEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import {
  Field,
  Said,
  TextArea,
  TextInput,
  Trouble,
} from "@/components/ui/field";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { Profile, ProfileLink } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { BIOGRAPHY_LIMIT } from "@/lib/profile-draft";
import { ProfileLinks } from "./ProfileLinks";
import { ProfilePicture } from "./ProfilePicture";
import { ProfilePreview } from "./ProfilePreview";

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

type Draft = {
  biography: string;
  contactEmail: string;
  displayName: string;
  links: ProfileLink[];
};

type Answer = Profile & { error?: string; field?: string };

function draftOf(profile: Profile): Draft {
  return {
    biography: profile.biography,
    contactEmail: profile.contactEmail,
    displayName: profile.displayName,
    links: profile.links,
  };
}

/** Everything a visitor meets at a creator's handle, written beside what it will look like. */
export function PublicProfileEditor() {
  const { account } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [said, setSaid] = useState("");
  const [trouble, setTrouble] = useState("");
  const [failedField, setFailedField] = useState("");
  const [fieldTrouble, setFieldTrouble] = useState("");
  const [saving, setSaving] = useState(false);
  const [picturePending, setPicturePending] = useState(false);
  const [confirmingRemoval, setConfirmingRemoval] = useState(false);
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

  const change = useCallback((field: keyof Draft, value: string) => {
    setDraft((current) => (current ? { ...current, [field]: value } : current));
  }, []);

  if (account === undefined || (account && !draft)) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        Reading your public profile…
      </p>
    );
  }

  if (!account) {
    return (
      <Gate
        action={
          <Button asChild size="large" variant="primary">
            <Link href="/sign-in">Sign in</Link>
          </Button>
        }
        body="Your public profile belongs to your account, so it needs a session."
        title="Sign in to edit your public profile"
      />
    );
  }

  if (!account.emailVerified) {
    return (
      <Gate
        action={
          <Button asChild size="large" variant="primary">
            <Link href="/verify-email">Verify email</Link>
          </Button>
        }
        body="Your handle stays public either way. The rest of the profile opens once an address is verified."
        title="Verify your email to publish a profile"
      />
    );
  }

  if (profile?.restricted) {
    return (
      <div className="max-w-[52ch] rounded-plate bg-stop-wash p-7">
        <ShieldOff
          aria-hidden="true"
          className="size-7 text-stop"
          strokeWidth={1.4}
        />
        <h2 className="mt-4 font-display text-section font-medium tracking-tight text-ink">
          Illarin has hidden what you added to your profile
        </h2>
        <p className="mt-3 font-prose text-prose text-ink">
          Your display name, picture, biography, contact address and links are
          not shown, and you cannot change them until Illarin restores them.
          Your handle, your account and your published work are untouched.
        </p>
        <p className="mt-3 font-prose text-prose text-ink">
          Write to{" "}
          <a
            className="font-medium text-stop underline underline-offset-4"
            href="mailto:team@illarin.xyz"
          >
            team@illarin.xyz
          </a>{" "}
          to have it looked at again.
        </p>
      </div>
    );
  }

  if (!draft || !profile) return null;

  function forget() {
    setSaid("");
    setTrouble("");
    setFailedField("");
    setFieldTrouble("");
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!draft) return;
    setSaving(true);
    forget();
    try {
      const response = await browserFetch("/api/v1/account/profile", {
        body: JSON.stringify(draft),
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        method: "PUT",
      });
      const answer = (await response.json()) as Answer;
      if (!response.ok) {
        const reason = answer.error ?? "The profile could not be saved.";
        setFailedField(answer.field ?? "");
        setFieldTrouble(answer.field ? reason : "");
        setTrouble(
          answer.field ? "Check the marked field and save again." : reason,
        );
        return;
      }
      setProfile(answer);
      setDraft(draftOf(answer));
      setSaid("Your public profile is saved.");
    } catch {
      setTrouble(UNREACHABLE);
    } finally {
      setSaving(false);
    }
  }

  async function writePicture(request: RequestInit, done: string) {
    setPicturePending(true);
    forget();
    try {
      const response = await browserFetch("/api/v1/account/profile/avatar", {
        credentials: "same-origin",
        ...request,
      });
      const answer = (await response.json()) as Answer;
      if (!response.ok) {
        setTrouble(answer.error ?? "That image could not be used.");
        return;
      }
      setProfile(answer);
      setSaid(done);
    } catch {
      setTrouble(UNREACHABLE);
    } finally {
      setPicturePending(false);
    }
  }

  const remaining = BIOGRAPHY_LIMIT - draft.biography.length;
  const failed = (field: string) => failedField === field;

  return (
    <div className="grid items-start gap-12 lg:grid-cols-[minmax(0,36rem)_minmax(0,22rem)] lg:gap-16">
      <form className="min-w-0" noValidate onSubmit={save}>
        <Group
          note="The face and the name a visitor meets first."
          title="Who you are"
        >
          <ProfilePicture
            confirmingRemoval={confirmingRemoval}
            onConfirmRemoval={setConfirmingRemoval}
            onRemove={() => {
              setConfirmingRemoval(false);
              void writePicture(
                { method: "DELETE" },
                "Your picture is removed.",
              );
            }}
            onUpload={(file) => {
              const body = new FormData();
              body.append("file", file);
              void writePicture(
                { body, method: "PUT" },
                "Your picture is updated.",
              );
            }}
            pending={picturePending}
            profile={profile}
          />

          <Field
            className="max-w-[22rem]"
            hint="Clearing it puts your handle back in its place."
            htmlFor="profile-display-name"
            label="Display name"
            trouble={failed("displayName") ? fieldTrouble : undefined}
          >
            <TextInput
              aria-describedby="profile-display-name-hint"
              aria-invalid={failed("displayName") || undefined}
              autoComplete="nickname"
              id="profile-display-name"
              maxLength={48}
              name="displayName"
              onChange={(event) => change("displayName", event.target.value)}
              placeholder={`@${profile.handle}`}
              type="text"
              value={draft.displayName}
            />
          </Field>
        </Group>

        <Group
          note="Up to 400 characters of plain text, under your name."
          title="What you say"
        >
          <Field
            hint={
              <span className="flex flex-wrap items-baseline justify-between gap-x-4">
                <span>No formatting.</span>
                <span
                  className={
                    remaining <= 40 ? "tabular-nums text-stop" : "tabular-nums"
                  }
                >
                  {remaining} left
                </span>
              </span>
            }
            htmlFor="profile-biography"
            label="Biography"
            trouble={failed("biography") ? fieldTrouble : undefined}
          >
            <TextArea
              aria-describedby="profile-biography-hint"
              aria-invalid={failed("biography") || undefined}
              id="profile-biography"
              maxLength={BIOGRAPHY_LIMIT}
              name="biography"
              onChange={(event) => change("biography", event.target.value)}
              rows={4}
              value={draft.biography}
            />
          </Field>
        </Group>

        <Group
          note="All of this is public. Your sign-in address is not, and is never copied here."
          title="Where to find you"
        >
          <Field
            className="max-w-[26rem]"
            hint="Leave it empty to show no address."
            htmlFor="profile-contact"
            label="Contact email"
            trouble={failed("contactEmail") ? fieldTrouble : undefined}
          >
            <TextInput
              aria-describedby="profile-contact-hint"
              aria-invalid={failed("contactEmail") || undefined}
              autoComplete="off"
              id="profile-contact"
              maxLength={254}
              name="contactEmail"
              onChange={(event) => change("contactEmail", event.target.value)}
              placeholder="nobody@example.com"
              type="email"
              value={draft.contactEmail}
            />
          </Field>

          <ProfileLinks
            links={draft.links}
            onChange={(links) =>
              setDraft((current) => (current ? { ...current, links } : current))
            }
            trouble={failed("links") ? fieldTrouble : undefined}
          />
        </Group>

        {trouble ? (
          <div className="mt-8">
            <Trouble>{trouble}</Trouble>
          </div>
        ) : null}

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <Button loading={saving} size="large" type="submit" variant="primary">
            {saving ? "Saving" : "Save profile"}
          </Button>
          <Button asChild variant="ghost">
            <Link href={`/@${profile.handle}`}>View public profile</Link>
          </Button>
          {said ? (
            <Said className="basis-full sm:basis-auto">{said}</Said>
          ) : null}
        </div>
      </form>

      <ProfilePreview
        biography={draft.biography}
        contactEmail={draft.contactEmail}
        displayName={draft.displayName}
        handle={profile.handle}
        links={draft.links}
        picture={profile.avatar}
      />
    </div>
  );
}

/** One thing the profile is made of, so the page is three questions rather than six boxes. */
function Group({
  children,
  note,
  title,
}: {
  children: ReactNode;
  note: string;
  title: string;
}) {
  return (
    <section className="border-t border-rule pt-6 first:border-0 first:pt-0 [&+section]:mt-10">
      <h2 className="font-display text-section font-medium tracking-tight text-ink">
        {title}
      </h2>
      <p className="mt-1 max-w-[52ch] font-prose text-ui text-mute">{note}</p>
      <div className="mt-6 grid gap-7">{children}</div>
    </section>
  );
}

function Gate({
  action,
  body,
  title,
}: {
  action: ReactNode;
  body: string;
  title: string;
}) {
  return (
    <div className="max-w-[42ch] rounded-plate bg-deep p-7">
      <ShieldCheck
        aria-hidden="true"
        className="size-7 text-accent"
        strokeWidth={1.4}
      />
      <h2 className="mt-4 font-display text-section font-medium tracking-tight text-ink">
        {title}
      </h2>
      <p className="mt-2 font-prose text-prose text-mute">{body}</p>
      <div className="mt-6">{action}</div>
    </div>
  );
}
