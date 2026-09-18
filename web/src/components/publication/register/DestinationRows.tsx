"use client";

import {
  CircleCheck,
  CircleDashed,
  CircleSlash,
  KeyRound,
  Plus,
  Power,
  SatelliteDish,
} from "lucide-react";
import { useState } from "react";
import {
  Mark,
  Nothing,
  PanelHead,
  Row,
  RowAction,
  RowMark,
  Rows,
  StartAction,
  type Tone,
} from "@/components/register/RowParts";
import {
  Consequence,
  StepAction,
  StepForm,
  StepNote,
} from "@/components/register/StepParts";
import { Field, TextInput } from "@/components/ui/field";
import {
  addChannel,
  addDestination,
  disableDestination,
  removeDestination,
  updateChannel,
  updateDestination,
  verifyDestination,
} from "@/lib/api/publication";
import type {
  AddedPublicationDestination,
  PublicationDestination,
  PublicationDestinationType,
  PublicationEvent,
} from "@/lib/api/query";
import {
  destinationActions,
  destinationStanding,
  destinationTakes,
  nothingIn,
} from "@/lib/publication-register";
import { DestinationKind, EventChoice } from "./choices";
import { TakeTheSecret } from "./SecretStep";

const STATE_MARKS: Record<
  PublicationDestination["state"],
  { icon: typeof CircleCheck; tone: Tone }
> = {
  active: { icon: CircleCheck, tone: "accent" },
  disabled: { icon: CircleSlash, tone: "quiet" },
  unverified: { icon: CircleDashed, tone: "quiet" },
};

export function DestinationRows({
  destinations,
  onFailure,
  onOpen,
  onSaved,
}: {
  destinations: PublicationDestination[];
  onFailure: (message: string) => void;
  onOpen: (destination: PublicationDestination | null) => void;
  onSaved: (saved: PublicationDestination) => void;
}) {
  const [working, setWorking] = useState("");

  async function prove(one: PublicationDestination) {
    setWorking(one.id);
    const answer = await verifyDestination(one.id);
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onSaved(answer.value);
  }

  return (
    <>
      <PanelHead
        action={
          <StartAction icon={Plus} onClick={() => onOpen(null)}>
            Add a destination
          </StartAction>
        }
        id="register-heading"
        title="Destinations"
      />

      {destinations.length === 0 ? (
        <Nothing>{nothingIn("destinations")}</Nothing>
      ) : (
        <Rows>
          {destinations.map((one) => {
            const state = STATE_MARKS[one.state];
            const open = destinationActions(one);
            return (
              <Row
                aside={
                  open.verify ? (
                    <RowAction
                      busy={working === one.id}
                      onClick={() => prove(one)}
                    >
                      {working === one.id ? "Verifying…" : "Verify"}
                    </RowAction>
                  ) : null
                }
                facts={
                  <>
                    <span>{one.host}</span>
                    <span>{destinationTakes(one)}</span>
                  </>
                }
                key={one.id}
                lead={
                  <RowMark tone={state.tone}>
                    <state.icon
                      aria-hidden="true"
                      className="size-5"
                      strokeWidth={1.8}
                    />
                  </RowMark>
                }
                onOpen={() => onOpen(one)}
                open={`Edit ${one.name}`}
                standing={destinationStanding(one)}
                title={one.name}
                trailing={
                  one.kind === "discord" ? <Mark>Discord</Mark> : undefined
                }
              />
            );
          })}
        </Rows>
      )}
    </>
  );
}

export function DestinationStep({
  existing,
  onClose,
  onFailure,
  onRemoved,
  onRotate,
  onSaved,
}: {
  existing: PublicationDestination | null;
  onClose: () => void;
  onFailure: (message: string) => void;
  onRemoved: () => void;
  onRotate: () => void;
  onSaved: (saved: PublicationDestination) => void;
}) {
  const [kind, setKind] = useState<PublicationDestinationType>(
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
  const [busy, setBusy] = useState(false);
  const [working, setWorking] = useState("");

  const open = existing
    ? destinationActions(existing)
    : { rotate: false, switchOff: false, verify: false };
  const ready = existing
    ? Boolean(name.trim())
    : Boolean(name.trim() && address.trim());

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
          address: address.trim() || undefined,
          events,
          name: name.trim(),
        })
      : addDestination({ address: address.trim(), events, name: name.trim() });
  }

  async function save() {
    setBusy(true);
    const written = await write();
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    if ("secret" in written.value) {
      onSaved(written.value.destination);
      setMade(written.value);
      return;
    }
    onSaved(written.value);
    onClose();
  }

  async function stand(
    what: "verify" | "switchOff",
    run: () => ReturnType<typeof verifyDestination>,
  ) {
    setWorking(what);
    const answer = await run();
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onSaved(answer.value);
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
      <TakeTheSecret
        carry="The receiver checks every request against this with HMAC-SHA256, in the webhook-signature header. Verify the endpoint once you have it in place."
        copied={copied}
        heading="Copy the signing secret now"
        line="Copy this secret now. You cannot view it again. If you lose it, rotate the signing secret."
        onCopied={setCopied}
        onDone={onClose}
        value={made.secret}
      />
    );
  }

  return (
    <StepForm
      busy={busy}
      commit={existing ? "Save" : "Add"}
      onCommit={save}
      ready={ready}
      under={
        existing ? (
          <Consequence
            action="Remove"
            busy={busy}
            confirm="Remove destination"
            onConfirm={remove}
          >
            Everything already sent to {existing.name} stays in the record.
            Anything still waiting stops.
          </Consequence>
        ) : null
      }
    >
      <StepNote>{hint(kind, Boolean(existing))}</StepNote>

      {existing ? (
        <div className="flex flex-wrap gap-2">
          {open.verify ? (
            <StepAction
              busy={working === "verify"}
              icon={SatelliteDish}
              onClick={() =>
                stand("verify", () => verifyDestination(existing.id))
              }
            >
              {working === "verify" ? "Verifying…" : "Verify endpoint"}
            </StepAction>
          ) : null}
          {open.switchOff ? (
            <StepAction
              busy={working === "switchOff"}
              icon={Power}
              onClick={() =>
                stand("switchOff", () => disableDestination(existing.id))
              }
            >
              Disable destination
            </StepAction>
          ) : null}
          {open.rotate ? (
            <StepAction icon={KeyRound} onClick={onRotate}>
              Rotate signing secret
            </StepAction>
          ) : null}
        </div>
      ) : null}

      {existing ? null : <DestinationKind chosen={kind} onChosen={setKind} />}

      {existing?.channel ? (
        <div className="rounded-plate bg-deep p-4">
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 font-ui text-meta">
            <dt className="text-mute">Server id</dt>
            <dd className="font-mono text-ink wrap-anywhere">
              {existing.channel.guildId}
            </dd>
            <dt className="text-mute">Channel id</dt>
            <dd className="font-mono text-ink wrap-anywhere">
              {existing.channel.channelId}
            </dd>
          </dl>
          <p className="mt-3 font-prose text-meta text-mute">
            Check these IDs against your Discord server and channel.
          </p>
        </div>
      ) : null}

      <Field
        hint="Shown to contributors when choosing announcements. The endpoint URL stays private."
        htmlFor="destination-name"
        label="Name"
      >
        <TextInput
          autoComplete="off"
          id="destination-name"
          maxLength={48}
          onChange={(event) => setName(event.target.value)}
          placeholder={kind === "discord" ? "Announcements" : "Release feed"}
          value={name}
        />
      </Field>

      <Field
        hint={addressHint(kind, existing)}
        htmlFor="destination-address"
        label="Webhook URL"
      >
        <TextInput
          autoComplete="off"
          id="destination-address"
          maxLength={300}
          onChange={(event) => setAddress(event.target.value)}
          placeholder={
            kind === "discord"
              ? "https://discord.com/api/webhooks/…"
              : "https://hooks.example.com/illarin"
          }
          type="url"
          value={address}
        />
      </Field>

      {kind === "discord" ? (
        <>
          <Field
            hint="The one role a writer may ask an announcement to mention. Leave both role fields empty to approve none."
            htmlFor="destination-role-id"
            label="Role id"
          >
            <TextInput
              autoComplete="off"
              className="font-mono"
              id="destination-role-id"
              inputMode="numeric"
              maxLength={20}
              onChange={(event) => setRoleId(event.target.value)}
              placeholder="000000000000000000"
              value={roleId}
            />
          </Field>
          <Field
            hint="Shown to contributors instead of the role ID."
            htmlFor="destination-role-name"
            label="Role name"
          >
            <TextInput
              autoComplete="off"
              id="destination-role-name"
              maxLength={48}
              onChange={(event) => setRoleName(event.target.value)}
              placeholder="Blog readers"
              value={roleName}
            />
          </Field>
        </>
      ) : (
        <EventChoice chosen={events} onChosen={setEvents} />
      )}
    </StepForm>
  );
}

function hint(kind: PublicationDestinationType, editing: boolean): string {
  if (kind === "discord") {
    return editing
      ? "Illarin masks the address after you save it. Leave it empty to keep the channel this destination already announces in."
      : "One Discord channel Illarin announces a post's first publication in. Illarin asks Discord what the address points at before saving it.";
  }
  return editing
    ? "The saved URL stays hidden. Verify a replacement endpoint before sending announcements to it."
    : "An endpoint for blog announcements. Verify it before sending announcements.";
}

function addressHint(
  kind: PublicationDestinationType,
  existing: PublicationDestination | null,
): string {
  if (existing) return `Now ${existing.address}. Leave this empty to keep it.`;
  if (kind === "discord") {
    return "The incoming webhook address Discord gives you for the channel. Illarin seals it and never shows it again.";
  }
  return "https on port 443, no username, password or fragment, and a host on the public internet.";
}
