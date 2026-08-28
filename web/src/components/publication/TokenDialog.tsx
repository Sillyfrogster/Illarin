"use client";

import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { issueToken } from "@/lib/api/publication";
import type { IssuedPublicationToken } from "@/lib/api/query";
import styles from "./TokenDialog.module.css";

export function TokenDialog({
  grantId,
  appName,
  onClose,
  onIssued,
  onFailure,
}: {
  grantId: string;
  appName: string;
  onClose: () => void;
  onIssued: (made: IssuedPublicationToken) => void;
  onFailure: (message: string) => void;
}) {
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState("");
  const [made, setMade] = useState<IssuedPublicationToken | null>(null);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);

  async function issue() {
    setBusy(true);
    const written = await issueToken(grantId, {
      name: name.trim(),
      expiresAt: expiry ? endOfDay(expiry) : undefined,
    });
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    setMade(written.value);
    onIssued(written.value);
  }

  if (made) {
    return (
      <FormDialog
        open
        title="Copy it now"
        hint="This is the only time Illarin can show you this token. Nothing here can read it back, so if it gets away from you, revoke it and make another."
        commit={copied ? "I have it" : "Close without copying"}
        acknowledge
        onClose={onClose}
        onCommit={onClose}
      >
        <Reveal value={made.value} copied={copied} onCopied={setCopied} />
      </FormDialog>
    );
  }

  return (
    <FormDialog
      open
      title="New token"
      hint={`One token for one tool. It publishes for ${appName} exactly as you can, and reaches nothing else on Illarin.`}
      commit="Make the token"
      busy={busy}
      ready={Boolean(name.trim())}
      onClose={onClose}
      onCommit={issue}
    >
      <Field
        label="What is it for"
        htmlFor="token-name"
        hint="Name the tool or machine that will carry it, so you know which one to revoke later."
      >
        <input
          id="token-name"
          value={name}
          placeholder="Release robot"
          autoComplete="off"
          maxLength={48}
          onChange={(event) => setName(event.target.value)}
        />
      </Field>
      <Field
        label="Stops working on"
        htmlFor="token-expiry"
        hint="Leave this empty and it works until you revoke it."
      >
        <input
          id="token-expiry"
          type="date"
          value={expiry}
          min={tomorrow()}
          onChange={(event) => setExpiry(event.target.value)}
        />
      </Field>
    </FormDialog>
  );
}

function Reveal({
  value,
  copied,
  onCopied,
}: {
  value: string;
  copied: boolean;
  onCopied: (copied: boolean) => void;
}) {
  const [trouble, setTrouble] = useState("");

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      onCopied(true);
    } catch {
      setTrouble("Your browser would not let us copy. Select it and copy it.");
    }
  }

  return (
    <div className={styles.reveal}>
      <code className={styles.value}>{value}</code>
      <button
        type="button"
        className={styles.copy}
        onClick={copy}
        aria-live="polite"
      >
        {copied ? (
          <Check size={15} strokeWidth={2} aria-hidden="true" />
        ) : (
          <Copy size={15} strokeWidth={1.8} aria-hidden="true" />
        )}
        {copied ? "Copied" : "Copy"}
      </button>
      <p className={styles.carry} aria-live="polite">
        {trouble ||
          "Send it as a bearer credential on the publication API. Keep it out of anything you commit or share."}
      </p>
    </div>
  );
}

function tomorrow() {
  const day = new Date();
  day.setDate(day.getDate() + 1);
  return day.toISOString().slice(0, 10);
}

function endOfDay(day: string) {
  return new Date(`${day}T23:59:59`).toISOString();
}
