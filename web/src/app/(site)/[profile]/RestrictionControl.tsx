"use client";

import { ShieldMinus, ShieldPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextArea, Trouble } from "@/components/ui/field";
import {
  fetchProfileRestriction,
  type ProfileRestriction,
  restoreProfile,
  restrictProfile,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

const REASON_LIMIT = 500;

const PANEL = "mt-8 rounded-plate bg-deep p-5 sm:p-6";
const HEADING = "flex items-center gap-2 font-ui text-ui font-medium text-ink";

/** Illarin's own control over a profile, which only an admin ever sees. */
export function RestrictionControl({
  handle,
  restricted,
}: {
  handle: string;
  restricted: boolean;
}) {
  const router = useRouter();
  const { account } = useAuth();
  const isAdmin = account?.role === "admin";
  const [inForce, setInForce] = useState<ProfileRestriction | null>(null);
  const [reading, setReading] = useState(false);
  const [composing, setComposing] = useState(false);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [confirmingRestore, setConfirmingRestore] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!isAdmin || !restricted) {
      setInForce(null);
      return;
    }
    let current = true;
    setReading(true);
    void fetchProfileRestriction(handle).then((found) => {
      if (!current) return;
      setInForce(found);
      setReading(false);
    });
    return () => {
      current = false;
    };
  }, [isAdmin, restricted, handle]);

  if (!isAdmin) return null;

  async function restrict(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const written = reason.trim();
    if (!written || pending) return;
    setPending(true);
    setMessage("");
    try {
      await restrictProfile(handle, written);
      setComposing(false);
      setReason("");
      router.refresh();
    } catch {
      setMessage("The profile could not be restricted. Try again.");
    } finally {
      setPending(false);
    }
  }

  async function restore() {
    if (pending) return;
    setPending(true);
    setConfirmingRestore(false);
    setMessage("");
    try {
      await restoreProfile(handle);
      router.refresh();
    } catch {
      setMessage("The profile could not be restored. Try again.");
    } finally {
      setPending(false);
    }
  }

  const failure = message ? (
    <div className="mt-4">
      <Trouble>{message}</Trouble>
    </div>
  ) : null;

  if (restricted) {
    return (
      <section aria-labelledby="restriction-heading" className={PANEL}>
        <h2 className={HEADING} id="restriction-heading">
          <ShieldMinus
            aria-hidden="true"
            className="size-4 text-stop"
            strokeWidth={1.7}
          />
          Restricted
        </h2>
        {inForce ? (
          <>
            <p className="mt-2 font-ui text-meta text-mute">
              {new Date(inForce.restrictedAt).toLocaleDateString(undefined, {
                day: "numeric",
                month: "long",
                year: "numeric",
              })}
              {inForce.restrictedBy ? ` by @${inForce.restrictedBy}` : null}
            </p>
            <p className="mt-3 max-w-[70ch] font-prose text-ui text-ink">
              {inForce.reason}
            </p>
          </>
        ) : (
          <p className="mt-3 max-w-[70ch] font-ui text-meta text-mute">
            {reading
              ? "Reading the reason…"
              : "The reason could not be read. Reload the page to see it."}
          </p>
        )}
        <div className="mt-5">
          {confirmingRestore ? (
            <>
              <p className="font-ui text-meta text-mute">
                Restore it? Everything the creator added becomes public again.
              </p>
              <div className="mt-3 flex flex-wrap gap-3">
                <Button loading={pending} onClick={restore} variant="primary">
                  Restore
                </Button>
                <Button
                  onClick={() => setConfirmingRestore(false)}
                  variant="ghost"
                >
                  Keep hidden
                </Button>
              </div>
            </>
          ) : (
            <Button
              disabled={pending}
              onClick={() => setConfirmingRestore(true)}
              variant="outline"
            >
              Restore profile
            </Button>
          )}
        </div>
        {failure}
      </section>
    );
  }

  if (!composing) {
    return (
      <div className="mt-8">
        <Button onClick={() => setComposing(true)} variant="ghost">
          <ShieldPlus aria-hidden="true" />
          Restrict profile
        </Button>
      </div>
    );
  }

  const remaining = REASON_LIMIT - reason.length;

  return (
    <form
      aria-labelledby="restriction-heading"
      className={PANEL}
      onSubmit={restrict}
    >
      <h2 className={HEADING} id="restriction-heading">
        <ShieldPlus
          aria-hidden="true"
          className="size-4 text-stop"
          strokeWidth={1.7}
        />
        Restrict this profile
      </h2>
      <p className="mt-2 max-w-[70ch] font-prose text-meta text-mute">
        The handle and the published work below stay. Everything the creator
        added to the profile is hidden until an admin restores it.
      </p>
      <div className="mt-5 max-w-[70ch]">
        <Field
          hint="Only admins read this. It is kept in the audit record."
          htmlFor="restriction-reason"
          label="Reason"
          trailing={
            <span
              className={cn(
                "font-ui text-meta tabular-nums",
                remaining <= 60 ? "text-stop" : "text-mute",
              )}
            >
              {remaining} left
            </span>
          }
        >
          <TextArea
            disabled={pending}
            id="restriction-reason"
            maxLength={REASON_LIMIT}
            onChange={(event) => setReason(event.target.value)}
            required
            rows={3}
            value={reason}
          />
        </Field>
      </div>
      <div className="mt-4 flex flex-wrap gap-3">
        <Button
          disabled={!reason.trim()}
          loading={pending}
          type="submit"
          variant="stop"
        >
          Restrict profile
        </Button>
        <Button
          onClick={() => {
            setComposing(false);
            setMessage("");
          }}
          variant="ghost"
        >
          Cancel
        </Button>
      </div>
      {failure}
    </form>
  );
}
