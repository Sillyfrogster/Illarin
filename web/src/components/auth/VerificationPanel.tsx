"use client";

import { Check, Mail } from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, Said, TextInput, Trouble } from "@/components/ui/field";
import { browserFetch } from "@/lib/api/browser-mutation";
import { useAuth } from "@/lib/auth";
import { safeInternalReturnPath } from "@/lib/internal-return";

type Standing = "checking" | "waiting" | "verified" | "refused";

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function VerificationPanel() {
  const search = useSearchParams();
  const router = useRouter();
  const { account, refresh } = useAuth();
  const token = search.get("token");
  const returnTo = safeInternalReturnPath(search.get("returnTo")) ?? "/browse";
  const returnLabel = returnTo.startsWith("/link")
    ? "Return to linking"
    : "Browse Illarin";
  const started = useRef(false);
  const [standing, setStanding] = useState<Standing>(
    token ? "checking" : "waiting",
  );
  const [said, setSaid] = useState("");
  const [changing, setChanging] = useState(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;

    async function verify() {
      try {
        const response = await browserFetch("/api/v1/auth/verify-email", {
          body: JSON.stringify({ token }),
          headers: { "Content-Type": "application/json" },
          method: "POST",
        });
        const answer = (await response.json()) as { error?: string };
        if (!response.ok) {
          setSaid(answer.error ?? "This verification link could not be used.");
          setStanding("refused");
          return;
        }
        await refresh();
        setStanding("verified");
        router.replace(
          returnTo === "/browse"
            ? "/verify-email?verified=1"
            : `/verify-email?verified=1&returnTo=${encodeURIComponent(returnTo)}`,
        );
      } catch {
        setSaid(
          "We could not reach Illarin. Check your connection and try the link again.",
        );
        setStanding("refused");
      }
    }

    void verify();
  }, [refresh, returnTo, router, token]);

  async function changeEmail(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    setChanging(true);
    setSaid("");

    try {
      const response = await browserFetch("/api/v1/account/email", {
        body: JSON.stringify({
          email: String(new FormData(form).get("email") ?? ""),
        }),
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        method: "PATCH",
      });
      const answer = (await response.json()) as { error?: string };
      if (!response.ok) {
        setSaid(answer.error ?? "The email address could not be changed.");
        return;
      }
      await refresh();
      form.reset();
      setSaid("A fresh verification link is on its way.");
    } catch {
      setSaid(UNREACHABLE);
    } finally {
      setChanging(false);
    }
  }

  if (standing === "checking") {
    return (
      <section aria-busy="true" aria-live="polite" className="max-w-[30rem]">
        <span className="grid size-12 place-items-center rounded-plate bg-accent-wash text-accent">
          <Mail aria-hidden="true" className="size-6" strokeWidth={1.5} />
        </span>
        <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
          Verifying your address
        </h2>
        <p className="mt-3 font-prose text-prose text-mute">
          Wait while Illarin checks your verification link.
        </p>
      </section>
    );
  }

  if (standing === "verified" || (!token && account?.emailVerified)) {
    return (
      <section aria-live="polite" className="max-w-[30rem]">
        <span className="grid size-12 place-items-center rounded-plate bg-accent-wash text-accent">
          <Check aria-hidden="true" className="size-6" strokeWidth={2} />
        </span>
        <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
          Your address is verified
        </h2>
        <p className="mt-3 font-prose text-prose text-mute">
          You can now publish assets and link applications.
        </p>
        <div className="mt-7">
          <Button asChild size="large" variant="primary">
            <Link href={returnTo}>{returnLabel}</Link>
          </Button>
        </div>
      </section>
    );
  }

  if (standing === "refused") {
    return (
      <section className="grid max-w-[30rem] gap-5">
        <div>
          <span className="grid size-12 place-items-center rounded-plate bg-stop-wash text-stop">
            <Mail aria-hidden="true" className="size-6" strokeWidth={1.5} />
          </span>
          <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
            Email verification failed
          </h2>
        </div>
        <Trouble>{said}</Trouble>
        <div>
          <Button asChild size="large" variant="primary">
            <Link href="/sign-in">Return to sign in</Link>
          </Button>
        </div>
      </section>
    );
  }

  return (
    <section className="grid max-w-[30rem] gap-6">
      <div>
        <span className="grid size-12 place-items-center rounded-plate bg-accent-wash text-accent">
          <Mail aria-hidden="true" className="size-6" strokeWidth={1.5} />
        </span>
        <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
          Check your email
        </h2>
        <p className="mt-3 font-prose text-prose text-mute">
          We sent a verification link
          {account?.email ? ` to ${account.email}` : " to your address"}. You
          can keep browsing while you wait.
        </p>
      </div>

      {account && !account.emailVerified ? (
        <form className="grid gap-3" onSubmit={changeEmail}>
          <Field htmlFor="corrected-email" label="Mistyped the address?">
            <div className="flex flex-wrap items-center gap-3">
              <TextInput
                autoComplete="email"
                className="min-w-0 flex-1 basis-56"
                id="corrected-email"
                name="email"
                placeholder="Correct email address"
                required
                type="email"
              />
              <Button loading={changing} type="submit" variant="secondary">
                {changing ? "Sending" : "Send a new link"}
              </Button>
            </div>
          </Field>
        </form>
      ) : null}

      {said ? <Said>{said}</Said> : null}

      <div>
        <Button asChild variant="ghost">
          <Link href="/browse">Browse while you wait</Link>
        </Button>
      </div>
    </section>
  );
}
