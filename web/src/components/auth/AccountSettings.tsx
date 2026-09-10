"use client";

import {
  Check,
  KeyRound,
  Mail,
  MessageCircle,
  ShieldCheck,
} from "lucide-react";
import Link from "next/link";
import { type FormEvent, type ReactNode, useState } from "react";
import { Button } from "@/components/ui/button";
import { Said, TextInput } from "@/components/ui/field";
import { type WayIn, type WayInId, waysIn } from "@/lib/account-access";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { SignedInAccount } from "@/lib/auth";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

type Refusal = { error?: string };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

const MARKS: Record<WayInId, ReactNode> = {
  discord: (
    <MessageCircle aria-hidden="true" className="size-5" strokeWidth={1.5} />
  ),
  email: <Mail aria-hidden="true" className="size-5" strokeWidth={1.5} />,
  password: (
    <KeyRound aria-hidden="true" className="size-5" strokeWidth={1.5} />
  ),
};

/** The independent ways back into this account, and what each one still needs. */
export function AccountSettings({ discordNotice }: { discordNotice?: string }) {
  const { account, setAccount } = useAuth();
  const [said, setSaid] = useState(discordNotice ?? "");
  const [passwordPending, setPasswordPending] = useState(false);
  const [detachPending, setDetachPending] = useState(false);

  if (account === undefined) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        Reading your sign-in methods…
      </p>
    );
  }

  if (!account) {
    return (
      <div className="max-w-[34rem] rounded-plate bg-deep p-7">
        <ShieldCheck
          aria-hidden="true"
          className="size-7 text-accent"
          strokeWidth={1.4}
        />
        <h3 className="mt-4 font-display text-section font-medium tracking-tight text-ink">
          Sign in to open account settings
        </h3>
        <p className="mt-2 font-prose text-prose text-mute">
          Sign in to view or change the ways you access your account.
        </p>
        <div className="mt-6">
          <Button asChild variant="primary">
            <Link href="/sign-in">Sign in</Link>
          </Button>
        </div>
      </div>
    );
  }

  async function setPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    setPasswordPending(true);
    setSaid("");

    try {
      const response = await browserFetch("/api/v1/account/password", {
        body: JSON.stringify({
          password: String(new FormData(form).get("password") ?? ""),
        }),
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        method: "PUT",
      });
      const answer = (await response.json()) as SignedInAccount & Refusal;
      if (!response.ok) {
        setSaid(answer.error ?? "The password could not be saved.");
        return;
      }
      setAccount(answer);
      form.reset();
      setSaid("Your email can now be used with this password.");
    } catch {
      setSaid(UNREACHABLE);
    } finally {
      setPasswordPending(false);
    }
  }

  async function detachDiscord() {
    setDetachPending(true);
    setSaid("");
    try {
      const response = await browserFetch("/api/v1/account/discord", {
        credentials: "same-origin",
        method: "DELETE",
      });
      const answer = (await response.json()) as SignedInAccount & Refusal;
      if (!response.ok) {
        setSaid(answer.error ?? "Discord could not be detached.");
        return;
      }
      setAccount(answer);
      setSaid("Discord is detached and free to be used on another account.");
    } catch {
      setSaid(UNREACHABLE);
    } finally {
      setDetachPending(false);
    }
  }

  const ways = waysIn(account);
  const settled = ways.filter((way) => way.settled).length;
  const discord = ways.find((way) => way.id === "discord");

  return (
    <div className="grid gap-5">
      <p className="font-ui text-ui text-mute">
        {settled === 0
          ? "Nothing gets you back into this account yet."
          : settled === 1
            ? "One way gets you back into this account. A second is worth having."
            : `${settled} independent ways get you back into this account.`}
      </p>

      {said ? <Said>{said}</Said> : null}

      <ul className="m-0 grid list-none gap-px overflow-hidden rounded-plate bg-rule p-0">
        {ways.map((way) => (
          <li className="bg-plane" key={way.id}>
            <div className="flex flex-wrap items-center gap-x-5 gap-y-4 px-5 py-5">
              <span
                className={cn(
                  "grid size-11 shrink-0 place-items-center rounded-control",
                  way.settled
                    ? "bg-accent-wash text-accent"
                    : "bg-deep text-mute",
                )}
              >
                {MARKS[way.id]}
              </span>
              <div className="min-w-0 flex-1 basis-56">
                <h3 className="font-ui text-ui font-medium text-ink">
                  {way.name}
                </h3>
                <p className="font-ui text-ui text-ink [overflow-wrap:anywhere]">
                  {way.standing}
                </p>
                <p className="mt-1 font-ui text-meta text-mute">{way.note}</p>
              </div>
              <WayAction
                account={account}
                detach={() => void detachDiscord()}
                detachPending={detachPending}
                way={way}
              />
            </div>

            {way.id === "password" && !way.settled ? (
              <form
                className="flex flex-wrap items-center gap-3 border-t border-rule px-5 py-4"
                noValidate
                onSubmit={setPassword}
              >
                <label className="sr-only" htmlFor="settings-password">
                  New password
                </label>
                <TextInput
                  autoComplete="new-password"
                  className="min-w-0 flex-1 basis-56"
                  id="settings-password"
                  name="password"
                  placeholder="Choose a password"
                  required
                  type="password"
                />
                <Button
                  loading={passwordPending}
                  type="submit"
                  variant="primary"
                >
                  {passwordPending ? "Saving" : "Add password"}
                </Button>
              </form>
            ) : null}
          </li>
        ))}
      </ul>

      {discord?.settled && !discord.canDetach ? (
        <p className="font-ui text-meta text-mute" id="detach-requirement">
          Keep at least one verified way in. Verify an email and add a password
          before detaching Discord.
        </p>
      ) : null}
    </div>
  );
}

/** The one thing a reader can do about a way in, or the mark saying it is done. */
function WayAction({
  account,
  detach,
  detachPending,
  way,
}: {
  account: SignedInAccount;
  detach: () => void;
  detachPending: boolean;
  way: WayIn;
}) {
  if (way.id === "email") {
    return account.emailVerified ? (
      <Settled />
    ) : (
      <Button asChild variant="secondary">
        <Link href="/verify-email">Add or verify email</Link>
      </Button>
    );
  }

  if (way.id === "discord") {
    return way.settled ? (
      <Button
        aria-describedby={way.canDetach ? undefined : "detach-requirement"}
        disabled={!way.canDetach}
        loading={detachPending}
        onClick={detach}
        variant="secondary"
      >
        {detachPending ? "Detaching" : "Detach Discord"}
      </Button>
    ) : (
      <Button asChild disabled={!way.canAttach} variant="secondary">
        <a
          aria-disabled={!way.canAttach}
          href="/api/v1/auth/discord?intent=attach"
          onClick={(event) => {
            if (!way.canAttach) event.preventDefault();
          }}
        >
          Attach Discord
        </a>
      </Button>
    );
  }

  return way.settled ? (
    <Button asChild variant="ghost">
      <Link href="/forgot-password">Reset password</Link>
    </Button>
  ) : null;
}

function Settled() {
  return (
    <span className="inline-flex min-h-11 items-center gap-2 font-ui text-meta font-medium text-accent">
      <Check aria-hidden="true" className="size-4" strokeWidth={2} />
      Verified
    </span>
  );
}
