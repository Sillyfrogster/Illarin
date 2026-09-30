"use client";

import { SiDiscord } from "@icons-pack/react-simple-icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, type MouseEvent, useRef, useState } from "react";
import {
  AdultContentChoice,
  AppChoice,
  adultNote,
} from "@/components/preferences/PreferenceChoices";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { CheckboxRow } from "@/components/ui/checkbox";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { type Refusal, readRefusal } from "@/lib/answer";
import { api } from "@/lib/api/client";
import type { AppName, NsfwPreference } from "@/lib/api/query";
import type { SignedInAccount } from "@/lib/auth";
import { useAuth } from "@/lib/auth";

const UNREACHABLE = "Can't reach Illarin. Check your connection and try again.";

const TERMS_REFUSAL = "Agree to the Terms to create an account.";

const legalLink =
  "font-medium text-accent underline underline-offset-4 hover:no-underline";

export function AccountForm({
  apps = [],
  discordError,
  initialApp = null,
  mode,
  returnTo,
}: {
  apps?: AppName[];
  discordError?: string;
  initialApp?: string | null;
  mode: "sign-in" | "sign-up";
  returnTo?: string;
}) {
  const router = useRouter();
  const { setAccount } = useAuth();
  const [refused, setRefused] = useState<Refusal | null>(null);
  const [pending, setPending] = useState(false);
  const [app, setApp] = useState(initialApp);
  const [adult, setAdult] = useState<NsfwPreference>("blurred");
  const [agreed, setAgreed] = useState(false);
  const terms = useRef<HTMLButtonElement>(null);

  const signUp = mode === "sign-up";
  const carry = returnTo ? `?returnTo=${encodeURIComponent(returnTo)}` : "";
  const discordQuery = new URLSearchParams({
    ...(returnTo ? { returnTo } : {}),
    ...(signUp && app ? { app } : {}),
    ...(signUp ? { nsfw: adult } : {}),
  }).toString();
  const discordHref = `/api/v1/auth/discord${discordQuery ? `?${discordQuery}` : ""}`;

  function refuseWithoutTerms(): boolean {
    if (!signUp || agreed) return false;
    setRefused({ error: TERMS_REFUSAL, field: "terms" });
    terms.current?.focus();
    return true;
  }

  function continueWithDiscord(event: MouseEvent<HTMLAnchorElement>) {
    if (refuseWithoutTerms()) event.preventDefault();
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (refuseWithoutTerms()) return;
    setPending(true);
    setRefused(null);

    const form = new FormData(event.currentTarget);
    const body: Record<string, string> = {
      email: String(form.get("email") ?? ""),
      password: String(form.get("password") ?? ""),
    };
    if (signUp) {
      body.handle = String(form.get("handle") ?? "");
      body.nsfwPreference = adult;
      if (app) body.app = app;
    }

    try {
      const { data, error } = await api<SignedInAccount>(
        "POST",
        signUp ? "/v1/auth/sign-up" : "/v1/auth/sign-in",
        { body },
      );
      if (!data) {
        setRefused(readRefusal(error));
        return;
      }
      setAccount(data);
      router.push(signUp ? `/verify-email${carry}` : (returnTo ?? "/browse"));
    } catch {
      setRefused({ error: UNREACHABLE });
    } finally {
      setPending(false);
    }
  }

  const failed = (field: string) => refused?.field === field;

  return (
    <form className="grid gap-7" noValidate onSubmit={submit}>
      {signUp ? (
        <>
          <fieldset className="min-w-0 border-0 p-0">
            <legend className="font-ui text-ui font-medium text-ink">
              Your app
            </legend>
            <p className="mt-1 font-ui text-meta text-mute">
              Browse shows only what works in your app. Change it in Settings.
            </p>
            <div className="mt-3.5">
              <AppChoice apps={apps} name="app" onChange={setApp} value={app} />
            </div>
          </fieldset>

          <fieldset className="min-w-0 border-0 p-0">
            <legend className="font-ui text-ui font-medium text-ink">
              Adult content
            </legend>
            <div className="mt-3">
              <AdultContentChoice
                name="nsfwPreference"
                onChange={setAdult}
                value={adult}
              />
            </div>
            <p className="mt-2.5 font-ui text-meta text-mute">
              {adultNote(adult)}
              <br />
              Blur and Show are for 18 and over.
            </p>
          </fieldset>
        </>
      ) : null}

      <div className="grid gap-5">
        {signUp ? (
          <div className="grid gap-2">
            <CheckboxRow
              aria-describedby={failed("terms") ? "terms-trouble" : undefined}
              aria-invalid={failed("terms") || undefined}
              checked={agreed}
              label={
                <>
                  I&rsquo;m 13 or older and agree to the{" "}
                  <Link
                    className={legalLink}
                    href="/legal/terms"
                    target="_blank"
                  >
                    Terms
                  </Link>{" "}
                  and the{" "}
                  <Link
                    className={legalLink}
                    href="/legal/privacy"
                    target="_blank"
                  >
                    Privacy Policy
                  </Link>
                </>
              }
              name="terms"
              onCheckedChange={(on) => {
                setAgreed(on);
                if (failed("terms")) setRefused(null);
              }}
              ref={terms}
            />
            {failed("terms") ? (
              <p
                className="pl-6.5 font-ui text-meta text-stop"
                id="terms-trouble"
              >
                {refused?.error}
              </p>
            ) : null}
          </div>
        ) : null}

        <Button asChild variant="secondary">
          <a href={discordHref} onClick={continueWithDiscord}>
            <SiDiscord aria-hidden="true" color="default" title="" />
            Continue with Discord
          </a>
        </Button>

        {discordError ? <Alert tone="stop">{discordError}</Alert> : null}

        {signUp ? null : (
          <p className="-mt-2 font-ui text-meta text-mute">
            By continuing, you agree to the{" "}
            <Link className={legalLink} href="/legal/terms">
              Terms
            </Link>{" "}
            and the{" "}
            <Link className={legalLink} href="/legal/privacy">
              Privacy Policy
            </Link>
            .
          </p>
        )}

        <p
          aria-hidden="true"
          className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 font-ui text-meta text-mute"
        >
          <span className="h-px bg-rule" />
          or use email
          <span className="h-px bg-rule" />
        </p>

        {signUp ? (
          <HandleField
            trouble={failed("handle") ? refused?.error : undefined}
          />
        ) : null}

        <Field
          htmlFor="account-email"
          label="Email"
          trouble={failed("email") ? refused?.error : undefined}
        >
          <Input
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
          <Input
            aria-invalid={failed("password") || undefined}
            autoComplete={signUp ? "new-password" : "current-password"}
            id="account-password"
            name="password"
            required
            type="password"
          />
        </Field>

        {refused?.error && !refused.field ? (
          <Alert tone="stop">{refused.error}</Alert>
        ) : null}

        <Button
          className="mt-1 shadow-none"
          loading={pending}
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
      </div>
    </form>
  );
}

function HandleField({ trouble }: { trouble?: string }) {
  return (
    <Field
      hint="3–32 lowercase letters, numbers, dots or underscores."
      htmlFor="account-handle"
      label="Handle"
      trouble={trouble}
    >
      <InputGroup>
        <InputGroupAddon>@</InputGroupAddon>
        <InputGroupInput
          autoCapitalize="none"
          autoComplete="username"
          id="account-handle"
          maxLength={32}
          minLength={3}
          name="handle"
          required
          spellCheck={false}
          type="text"
        />
      </InputGroup>
    </Field>
  );
}
