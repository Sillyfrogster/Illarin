"use client";

import { MessageCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  controlClasses,
  Field,
  TextInput,
  Trouble,
} from "@/components/ui/field";
import type { Refusal } from "@/lib/answer";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { SignedInAccount } from "@/lib/auth";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function AccountForm({
  discordError,
  mode,
  returnTo,
}: {
  discordError?: string;
  mode: "sign-in" | "sign-up";
  returnTo?: string;
}) {
  const router = useRouter();
  const { setAccount } = useAuth();
  const [refused, setRefused] = useState<Refusal | null>(null);
  const [pending, setPending] = useState(false);

  const signUp = mode === "sign-up";
  const carry = returnTo ? `?returnTo=${encodeURIComponent(returnTo)}` : "";

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setRefused(null);

    const form = new FormData(event.currentTarget);
    const body: Record<string, string> = {
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
    };
    if (signUp) body.handle = String(form.get("handle") ?? "");

    try {
      const response = await browserFetch(
        signUp ? "/api/v1/auth/sign-up" : "/api/v1/auth/sign-in",
        {
          body: JSON.stringify(body),
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          method: "POST",
        },
      );
      const answer = (await response.json()) as SignedInAccount & Refusal;
      if (!response.ok) {
        setRefused(answer);
        return;
      }
      setAccount(answer);
      router.push(signUp ? `/verify-email${carry}` : (returnTo ?? "/browse"));
    } catch {
      setRefused({ error: UNREACHABLE });
    } finally {
      setPending(false);
    }
  }

  const failed = (field: string) => refused?.field === field;

  return (
    <form className="grid max-w-[30rem] gap-5" noValidate onSubmit={submit}>
      <Button asChild size="large" variant="outline">
        <a href={`/api/v1/auth/discord${carry}`}>
          <MessageCircle aria-hidden="true" />
          Continue with Discord
        </a>
      </Button>

      {discordError ? <Trouble>{discordError}</Trouble> : null}

      <p
        aria-hidden="true"
        className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 font-ui text-meta text-mute"
      >
        <span className="h-px bg-rule" />
        or use email
        <span className="h-px bg-rule" />
      </p>

      {signUp ? (
        <Field
          hint="3–32 lowercase letters, numbers, dots or underscores."
          htmlFor="account-handle"
          label="Handle"
          trouble={failed("handle") ? refused?.error : undefined}
        >
          <div
            className={cn(
              controlClasses,
              "flex items-center gap-0.5 px-0 py-0",
              failed("handle") && "inset-ring-2 inset-ring-stop",
            )}
          >
            <span
              aria-hidden="true"
              className="pl-3.5 font-display text-lede text-mute"
            >
              @
            </span>
            <input
              aria-describedby="account-handle-hint"
              aria-invalid={failed("handle") || undefined}
              autoCapitalize="none"
              autoComplete="username"
              className="min-h-11 w-full min-w-0 rounded-control bg-transparent pr-3.5 font-ui text-ui text-ink outline-offset-2"
              id="account-handle"
              maxLength={32}
              minLength={3}
              name="handle"
              required
              spellCheck={false}
              type="text"
            />
          </div>
        </Field>
      ) : null}

      <Field
        htmlFor="account-email"
        label="Email address"
        trouble={failed("email") ? refused?.error : undefined}
      >
        <TextInput
          aria-invalid={failed("email") || undefined}
          autoCapitalize="none"
          autoComplete="email"
          id="account-email"
          name="email"
          required
          spellCheck={false}
          type="email"
        />
      </Field>

      <Field
        htmlFor="account-password"
        label="Password"
        trailing={
          signUp ? null : (
            <Link
              className="-mx-1 inline-flex min-h-11 items-center px-1 font-ui text-meta font-medium text-accent underline-offset-4 hover:underline"
              href="/forgot-password"
            >
              Forgot password?
            </Link>
          )
        }
        trouble={failed("password") ? refused?.error : undefined}
      >
        <TextInput
          aria-invalid={failed("password") || undefined}
          autoComplete={signUp ? "new-password" : "current-password"}
          id="account-password"
          name="password"
          required
          type="password"
        />
      </Field>

      {refused?.error && !refused.field ? (
        <Trouble>{refused.error}</Trouble>
      ) : null}

      <Button
        className="mt-1 shadow-none"
        loading={pending}
        size="large"
        type="submit"
        variant="primary"
      >
        {pending
          ? signUp
            ? "Creating your account"
            : "Signing you in"
          : signUp
            ? "Create account"
            : "Sign in"}
      </Button>

      <p className="flex flex-wrap items-center gap-x-1.5 font-ui text-ui text-mute">
        {signUp ? "Already have an account?" : "New to Illarin?"}
        <Link
          className="-mx-1 inline-flex min-h-11 items-center px-1 font-medium text-accent underline-offset-4 hover:underline"
          href={`${signUp ? "/sign-in" : "/sign-up"}${carry}`}
        >
          {signUp ? "Sign in" : "Create an account"}
        </Link>
      </p>
    </form>
  );
}
