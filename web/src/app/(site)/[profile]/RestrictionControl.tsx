"use client";

import { ShieldMinus, ShieldPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useEffect, useState } from "react";
import {
  fetchProfileRestriction,
  type ProfileRestriction,
  restoreProfile,
  restrictProfile,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import styles from "./RestrictionControl.module.css";

const REASON_LIMIT = 500;

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
    void fetchProfileRestriction(handle).then((found) => {
      if (current) setInForce(found);
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

  if (restricted) {
    return (
      <section className={styles.staff} aria-labelledby="restriction-heading">
        <h2 className={styles.heading} id="restriction-heading">
          <ShieldMinus size={15} strokeWidth={1.7} aria-hidden="true" />
          Restricted
        </h2>
        {inForce ? (
          <>
            <p className={styles.note}>
              {new Date(inForce.restrictedAt).toLocaleDateString(undefined, {
                day: "numeric",
                month: "long",
                year: "numeric",
              })}
              {inForce.restrictedBy ? ` by @${inForce.restrictedBy}` : null}
            </p>
            <p className={styles.reason}>{inForce.reason}</p>
          </>
        ) : null}
        {confirmingRestore ? (
          <div className={styles.confirm}>
            <p className={styles.note}>
              Restore it? Everything the creator added becomes public again.
            </p>
            <div className={styles.commit}>
              <button
                type="button"
                className={styles.action}
                onClick={restore}
                disabled={pending}
              >
                {pending ? "Restoring…" : "Restore"}
              </button>
              <button
                type="button"
                className={styles.quiet}
                onClick={() => setConfirmingRestore(false)}
              >
                Keep hidden
              </button>
            </div>
          </div>
        ) : (
          <button
            type="button"
            className={styles.action}
            onClick={() => setConfirmingRestore(true)}
            disabled={pending}
          >
            Restore profile
          </button>
        )}
        {message ? (
          <p className={styles.failure} role="alert">
            {message}
          </p>
        ) : null}
      </section>
    );
  }

  const remaining = REASON_LIMIT - reason.length;

  if (!composing) {
    return (
      <button
        type="button"
        className={styles.open}
        onClick={() => setComposing(true)}
      >
        <ShieldPlus size={15} strokeWidth={1.7} aria-hidden="true" />
        Restrict profile
      </button>
    );
  }

  return (
    <form
      className={styles.staff}
      onSubmit={restrict}
      aria-labelledby="restriction-heading"
    >
      <h2 className={styles.heading} id="restriction-heading">
        <ShieldPlus size={15} strokeWidth={1.7} aria-hidden="true" />
        Restrict this profile
      </h2>
      <p className={styles.note}>
        The handle and the published work below stay. Everything the creator
        added to the profile is hidden until an admin restores it.
      </p>
      <label htmlFor="restriction-reason">Reason</label>
      <textarea
        id="restriction-reason"
        rows={3}
        maxLength={REASON_LIMIT}
        value={reason}
        disabled={pending}
        onChange={(event) => setReason(event.target.value)}
        required
      />
      <p className={styles.note}>
        <span>Only admins read this. It is kept in the audit record.</span>
        <span className={styles.count} data-low={remaining <= 60 || undefined}>
          {remaining} left
        </span>
      </p>
      <div className={styles.commit}>
        <button
          type="submit"
          className={styles.critical}
          disabled={pending || !reason.trim()}
        >
          {pending ? "Restricting…" : "Restrict profile"}
        </button>
        <button
          type="button"
          className={styles.quiet}
          onClick={() => {
            setComposing(false);
            setMessage("");
          }}
        >
          Cancel
        </button>
      </div>
      {message ? (
        <p className={styles.failure} role="alert">
          {message}
        </p>
      ) : null}
    </form>
  );
}
