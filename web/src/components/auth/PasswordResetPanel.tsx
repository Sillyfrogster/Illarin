"use client";

import { Check, KeyRound, Mail } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { type FormEvent, type ReactNode, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextInput, Trouble } from "@/components/ui/field";
import { refusalMessage } from "@/lib/answer";
import { browserFetch } from "@/lib/api/browser-mutation";

type Result = { ok: true } | { ok: false; error: string };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

async function post(
  endpoint: string,
  body: Record<string, string>,
  fallback: string,
): Promise<Result> {
  try {
    const response = await browserFetch(endpoint, {
      body: JSON.stringify(body),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    });
    if (response.ok) return { ok: true };
    return {
      error: refusalMessage(await response.json(), fallback),
      ok: false,
    };
  } catch {
    return { error: UNREACHABLE, ok: false };
  }
}

/** Where a recovery step has landed, said once and acted on once. */
function Landing({
  action,
  body,
  mark,
  spoken = false,
  title,
}: {
  action: ReactNode;
  body: string;
  mark: ReactNode;
  spoken?: boolean;
  title: string;
}) {
  return (
    <section
      aria-live={spoken ? "polite" : undefined}
      className="max-w-[30rem]"
    >
      {mark}
      <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
        {title}
      </h2>
      <p className="mt-3 font-prose text-prose text-mute">{body}</p>
      <div className="mt-7">{action}</div>
    </section>
  );
}

function Mark({ children }: { children: ReactNode }) {
  return (
    <span className="grid size-12 place-items-center rounded-plate bg-accent-wash text-accent">
      {children}
    </span>
  );
}

/** Asks where to send a one-use link, and never says whether the address was found. */
export function PasswordResetRequestPanel() {
  const [pending, setPending] = useState(false);
  const [sent, setSent] = useState(false);
  const [trouble, setTrouble] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setTrouble("");
    const form = new FormData(event.currentTarget);

    const result = await post(
      "/api/v1/auth/password-reset",
      { email: String(form.get("email") ?? "") },
      "The reset request could not be sent.",
    );
    if (result.ok) setSent(true);
    else setTrouble(result.error);
    setPending(false);
  }

  if (sent) {
    return (
      <Landing
        action={
          <Button asChild size="large" variant="primary">
            <Link href="/sign-in">Return to sign in</Link>
          </Button>
        }
        body="If that verified address belongs to an account, a one-use password link is on its way."
        mark={
          <Mark>
            <Mail aria-hidden="true" className="size-6" strokeWidth={1.5} />
          </Mark>
        }
        spoken
        title="Check your email"
      />
    );
  }

  return (
    <form className="grid max-w-[30rem] gap-6" noValidate onSubmit={submit}>
      <Field
        hint="This also works if Discord has been your only way into Illarin until now."
        htmlFor="reset-email"
        label="Verified email address"
      >
        <TextInput
          aria-describedby="reset-email-hint"
          autoCapitalize="none"
          autoComplete="email"
          id="reset-email"
          name="email"
          required
          spellCheck={false}
          type="email"
        />
      </Field>

      {trouble ? <Trouble>{trouble}</Trouble> : null}

      <div className="flex flex-wrap items-center gap-3">
        <Button loading={pending} size="large" type="submit" variant="primary">
          {pending ? "Sending a reset link" : "Send reset link"}
        </Button>
        <Button asChild variant="ghost">
          <Link href="/sign-in">Return to sign in</Link>
        </Button>
      </div>
    </form>
  );
}

/** Takes the new password the one-use link opened the door for. */
export function PasswordResetCompletionPanel() {
  const token = useSearchParams().get("token");
  const [pending, setPending] = useState(false);
  const [complete, setComplete] = useState(false);
  const [trouble, setTrouble] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    setPending(true);
    setTrouble("");
    const form = new FormData(event.currentTarget);

    const result = await post(
      "/api/v1/auth/password-reset/complete",
      { password: String(form.get("password") ?? ""), token },
      "This password reset link could not be used.",
    );
    if (result.ok) setComplete(true);
    else setTrouble(result.error);
    setPending(false);
  }

  if (complete) {
    return (
      <Landing
        action={
          <Button asChild size="large" variant="primary">
            <Link href="/sign-in">Sign in with email</Link>
          </Button>
        }
        body="You can now return with your verified email, even without Discord."
        mark={
          <Mark>
            <Check aria-hidden="true" className="size-6" strokeWidth={2} />
          </Mark>
        }
        spoken
        title="Your password is ready"
      />
    );
  }

  if (!token) {
    return (
      <Landing
        action={
          <Button asChild size="large" variant="primary">
            <Link href="/forgot-password">Request another link</Link>
          </Button>
        }
        body="Request a fresh password link and open it from your email."
        mark={
          <Mark>
            <KeyRound aria-hidden="true" className="size-6" strokeWidth={1.5} />
          </Mark>
        }
        title="This link is incomplete"
      />
    );
  }

  return (
    <form className="grid max-w-[30rem] gap-6" noValidate onSubmit={submit}>
      <Field
        hint="The link can be used once. Your new password may be any length."
        htmlFor="reset-password"
        label="New password"
      >
        <TextInput
          aria-describedby="reset-password-hint"
          autoComplete="new-password"
          id="reset-password"
          name="password"
          required
          type="password"
        />
      </Field>

      {trouble ? <Trouble>{trouble}</Trouble> : null}

      <Button loading={pending} size="large" type="submit" variant="primary">
        {pending ? "Setting your password" : "Set password"}
      </Button>
    </form>
  );
}
