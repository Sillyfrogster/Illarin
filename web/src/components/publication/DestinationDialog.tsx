"use client";

import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { RevealOnce } from "@/components/console/RevealOnce";
import {
  addDestination,
  removeDestination,
  updateDestination,
} from "@/lib/api/publication";
import type {
  AddedPublicationDestination,
  PublicationDestination,
  PublicationEvent,
} from "@/lib/api/query";
import styles from "./AppDialog.module.css";
import { EventChoice } from "./EventChoice";

export function DestinationDialog({
  existing,
  onClose,
  onSaved,
  onRemoved,
  onFailure,
}: {
  existing: PublicationDestination | null;
  onClose: () => void;
  onSaved: (saved: PublicationDestination, added: boolean) => void;
  onRemoved: () => void;
  onFailure: (message: string) => void;
}) {
  const [name, setName] = useState(existing?.name ?? "");
  const [address, setAddress] = useState("");
  const [events, setEvents] = useState<PublicationEvent[]>(
    existing?.events ?? ["publication.post.published.v1"],
  );
  const [made, setMade] = useState<AddedPublicationDestination | null>(null);
  const [copied, setCopied] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const ready = existing
    ? Boolean(name.trim())
    : Boolean(name.trim() && address.trim());

  async function save() {
    setBusy(true);
    if (!existing) {
      const written = await addDestination({
        name: name.trim(),
        address: address.trim(),
        events,
      });
      setBusy(false);
      if (written.error || !written.value) {
        onFailure(written.error ?? "");
        return;
      }
      onSaved(written.value.destination, true);
      setMade(written.value);
      return;
    }
    const written = await updateDestination(existing.id, {
      name: name.trim(),
      address: address.trim() || undefined,
      events,
    });
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    onSaved(written.value, false);
    onClose();
  }

  async function remove() {
    if (!existing) return;
    setBusy(true);
    const answer = await removeDestination(existing.id);
    setBusy(false);
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    onRemoved();
    onClose();
  }

  if (made) {
    return (
      <FormDialog
        open
        acknowledge
        title="Copy the signing secret now"
        hint="This is the only time Illarin can show it. Nothing here can read it back, so if it gets away from you, point the destination somewhere else and start again."
        commit={copied ? "I have it" : "Close without copying"}
        onClose={onClose}
        onCommit={onClose}
      >
        <RevealOnce
          carry="The receiver checks every request against this with HMAC-SHA256, in the webhook-signature header. Verify the endpoint once you have it in place."
          copied={copied}
          onCopied={setCopied}
          value={made.secret}
        />
      </FormDialog>
    );
  }

  return (
    <FormDialog
      open
      title={existing ? "Edit this destination" : "Add a destination"}
      hint={
        existing
          ? "Illarin masks the address after you save it. Changing it needs the new endpoint to prove it is listening before anything is sent there."
          : "One endpoint Illarin sends published posts to. It receives nothing until it proves it is listening."
      }
      commit={existing ? "Save" : "Add"}
      busy={busy}
      ready={ready}
      onClose={onClose}
      onCommit={save}
      destructive={
        existing ? (
          <button
            type="button"
            className={styles.retire}
            onClick={() => (confirming ? remove() : setConfirming(true))}
          >
            {confirming ? "Remove, and stop what is waiting" : "Remove"}
          </button>
        ) : null
      }
    >
      <Field
        label="Name"
        htmlFor="destination-name"
        hint="What a writer picks from. They never see the address."
      >
        <input
          id="destination-name"
          value={name}
          maxLength={48}
          placeholder="Release feed"
          autoComplete="off"
          onChange={(event) => setName(event.target.value)}
        />
      </Field>

      <Field
        label="Address"
        htmlFor="destination-address"
        hint={
          existing
            ? `Now ${existing.address}. Leave this empty to keep it.`
            : "https on port 443, no username, password or fragment, and a host on the public internet."
        }
      >
        <input
          id="destination-address"
          type="url"
          value={address}
          maxLength={300}
          placeholder="https://hooks.example.com/illarin"
          autoComplete="off"
          onChange={(event) => setAddress(event.target.value)}
        />
      </Field>

      <EventChoice chosen={events} onChosen={setEvents} />

      {confirming && existing ? (
        <p className={styles.warning} role="alert">
          Everything already sent to {existing.name} stays in the record.
          Anything still waiting stops. Press remove again to confirm.
        </p>
      ) : null}
    </FormDialog>
  );
}
