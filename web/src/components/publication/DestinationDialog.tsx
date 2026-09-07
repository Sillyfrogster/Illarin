"use client";

import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { RevealOnce } from "@/components/console/RevealOnce";
import {
  addChannel,
  addDestination,
  removeDestination,
  updateChannel,
  updateDestination,
} from "@/lib/api/publication";
import type {
  AddedPublicationDestination,
  PublicationDestination,
  PublicationDestinationKind,
  PublicationEvent,
} from "@/lib/api/query";
import styles from "./AppDialog.module.css";
import own from "./DestinationDialog.module.css";
import { DestinationKind } from "./DestinationKind";
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
  const [kind, setKind] = useState<PublicationDestinationKind>(
    existing?.kind ?? "discord",
  );
  const [name, setName] = useState(existing?.name ?? "");
  const [address, setAddress] = useState("");
  const [roleId, setRoleId] = useState(existing?.channel?.roleId ?? "");
  const [roleName, setRoleName] = useState(existing?.channel?.roleName ?? "");
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
    const written = await write();
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    if ("secret" in written.value) {
      onSaved(written.value.destination, true);
      setMade(written.value);
      return;
    }
    onSaved(written.value, !existing);
    onClose();
  }

  function write() {
    if (kind === "discord") {
      const channel = {
        name: name.trim(),
        roleId: roleId.trim(),
        roleName: roleName.trim(),
      };
      return existing
        ? updateChannel(existing.id, {
            ...channel,
            address: address.trim() || undefined,
          })
        : addChannel({ ...channel, address: address.trim() });
    }
    return existing
      ? updateDestination(existing.id, {
          name: name.trim(),
          address: address.trim() || undefined,
          events,
        })
      : addDestination({ name: name.trim(), address: address.trim(), events });
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
      hint={hint(kind, Boolean(existing))}
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
      {existing ? null : <DestinationKind chosen={kind} onChosen={setKind} />}

      {existing?.channel ? (
        <div className={own.confirmed}>
          <dl className={own.channel}>
            <dt>Server id</dt>
            <dd>{existing.channel.guildId}</dd>
            <dt>Channel id</dt>
            <dd>{existing.channel.channelId}</dd>
          </dl>
          <p className={own.compare}>
            What Discord answered with. Compare them against the channel in
            Discord to be sure this is the right one.
          </p>
        </div>
      ) : null}

      <Field
        label="Name"
        htmlFor="destination-name"
        hint="What a writer picks from. They never see the address."
      >
        <input
          id="destination-name"
          value={name}
          maxLength={48}
          placeholder={kind === "discord" ? "Announcements" : "Release feed"}
          autoComplete="off"
          onChange={(event) => setName(event.target.value)}
        />
      </Field>

      <Field
        label="Address"
        htmlFor="destination-address"
        hint={addressHint(kind, existing)}
      >
        <input
          id="destination-address"
          type="url"
          value={address}
          maxLength={300}
          placeholder={
            kind === "discord"
              ? "https://discord.com/api/webhooks/…"
              : "https://hooks.example.com/illarin"
          }
          autoComplete="off"
          onChange={(event) => setAddress(event.target.value)}
        />
      </Field>

      {kind === "discord" ? (
        <div className={own.role}>
          <Field
            label="Role id"
            htmlFor="destination-role-id"
            hint="The one role a writer may ask an announcement to mention. Leave both role fields empty to approve none."
          >
            <input
              id="destination-role-id"
              value={roleId}
              maxLength={20}
              inputMode="numeric"
              placeholder="000000000000000000"
              autoComplete="off"
              onChange={(event) => setRoleId(event.target.value)}
            />
          </Field>

          <Field
            label="Role name"
            htmlFor="destination-role-name"
            hint="What a writer sees instead of the id."
          >
            <input
              id="destination-role-name"
              value={roleName}
              maxLength={48}
              placeholder="Blog readers"
              autoComplete="off"
              onChange={(event) => setRoleName(event.target.value)}
            />
          </Field>
        </div>
      ) : (
        <EventChoice chosen={events} onChosen={setEvents} />
      )}

      {confirming && existing ? (
        <p className={styles.warning} role="alert">
          Everything already sent to {existing.name} stays in the record.
          Anything still waiting stops. Press remove again to confirm.
        </p>
      ) : null}
    </FormDialog>
  );
}

function hint(kind: PublicationDestinationKind, editing: boolean): string {
  if (kind === "discord") {
    return editing
      ? "Illarin masks the address after you save it. Leave it empty to keep the channel this destination already announces in."
      : "One Discord channel Illarin announces a post's first publication in. Illarin asks Discord what the address points at before saving it.";
  }
  return editing
    ? "Illarin masks the address after you save it. Changing it needs the new endpoint to prove it is listening before anything is sent there."
    : "One endpoint Illarin sends published posts to. It receives nothing until it proves it is listening.";
}

function addressHint(
  kind: PublicationDestinationKind,
  existing: PublicationDestination | null,
): string {
  if (existing) return `Now ${existing.address}. Leave this empty to keep it.`;
  if (kind === "discord") {
    return "The incoming webhook address Discord gives you for the channel. Illarin seals it and never shows it again.";
  }
  return "https on port 443, no username, password or fragment, and a host on the public internet.";
}
