"use client";

import { motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Consequence, StepNote } from "@/components/register/StepParts";
import { Button } from "@/components/ui/button";
import { Field, Said, TextInput, Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { RevealOnce } from "@/components/ui/reveal-once";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import {
  type AssetUpdateDestination,
  addUpdateDestination,
  changeUpdateDestination,
  disableUpdateDestination,
  removeUpdateDestination,
  rotateUpdateDestinationSecret,
  verifyUpdateDestination,
} from "@/lib/api/asset-destinations";
import { cn } from "@/lib/cn";

export function DestinationEditor({
  existing,
  busy,
  onBusy,
  onSaved,
  onRemoved,
  onClose,
}: {
  existing: AssetUpdateDestination | null;
  busy: boolean;
  onBusy: (busy: boolean) => void;
  onSaved: (destination: AssetUpdateDestination) => void;
  onRemoved: (id: string) => void;
  onClose: () => void;
}) {
  const [current, setCurrent] = useState(existing);
  const [kind, setKind] = useState<AssetUpdateDestination["kind"]>(
    existing?.kind ?? "discord",
  );
  const [name, setName] = useState(existing?.name ?? "");
  const [address, setAddress] = useState("");
  const [secret, setSecret] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [working, setWorking] = useState("");
  const heading = useRef<HTMLHeadingElement>(null);
  const still = useReducedMotion();

  useEffect(() => {
    heading.current?.focus();
  }, []);

  function start(action: string) {
    onBusy(true);
    setWorking(action);
    setError("");
    setNotice("");
  }

  function finish() {
    onBusy(false);
    setWorking("");
  }

  function receive(saved: AssetUpdateDestination) {
    setCurrent(saved);
    onSaved(saved);
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    start("save");
    const answer = current
      ? await changeUpdateDestination(current.id, {
          name: name.trim(),
          address: address.trim() || undefined,
        })
      : await addUpdateDestination({
          name: name.trim(),
          kind,
          address: address.trim(),
        });
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not save the destination.");
      return;
    }
    setAddress("");
    if ("destination" in answer.value) {
      receive(answer.value.destination);
      setSecret(answer.value.secret ?? "");
      setCopied(false);
    } else receive(answer.value);
    setNotice(
      kind === "discord" ? "Discord channel saved." : "Destination saved.",
    );
  }

  async function verify() {
    if (!current || busy) return;
    start("verify");
    const answer = await verifyUpdateDestination(current.id);
    finish();
    if (!answer.value) {
      setError(
        answer.error ||
          "Verification failed. Check the receiver and try again.",
      );
      return;
    }
    receive(answer.value);
    setNotice("Verified. This destination is ready to select for your assets.");
  }

  async function disable() {
    if (!current || busy) return;
    start("disable");
    const answer = await disableUpdateDestination(current.id);
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not disable the destination.");
      return;
    }
    receive(answer.value);
    setNotice("Destination disabled.");
  }

  async function rotate() {
    if (!current || busy) return;
    start("rotate");
    const answer = await rotateUpdateDestinationSecret(current.id);
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not rotate the signing secret.");
      return;
    }
    receive(answer.value.destination);
    setSecret(answer.value.secret ?? "");
    setCopied(false);
    setNotice(
      "Signing secret rotated. The previous secret remains valid for 24 hours.",
    );
  }

  async function remove() {
    if (!current || busy) return;
    start("remove");
    const answer = await removeUpdateDestination(current.id);
    finish();
    if (answer.error) {
      setError(answer.error);
      return;
    }
    onRemoved(current.id);
  }

  return (
    <motion.section
      aria-labelledby="destination-editor"
      className="min-w-0 rounded-plate bg-deep p-6 lg:self-start lg:p-8"
      initial={{ opacity: 0, y: still ? 0 : 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: still ? 0 : 0.22 }}
    >
      <div className="mb-6 flex items-start justify-between gap-4">
        <h2
          className="font-display text-section font-medium text-ink outline-offset-3"
          id="destination-editor"
          ref={heading}
          tabIndex={-1}
        >
          {current ? "Destination settings" : "Connect a destination"}
        </h2>
        <Button
          aria-label="Close destination settings"
          disabled={busy}
          onClick={onClose}
          size="icon"
          variant="ghost"
        >
          <X aria-hidden="true" className="size-5" />
        </Button>
      </div>
      <div className="grid gap-5">
        {error ? <Trouble>{error}</Trouble> : null}
        {notice ? <Said>{notice}</Said> : null}
        {secret ? (
          <section aria-labelledby="save-signing-secret" className="grid gap-3">
            <h3
              className="font-ui text-ui font-medium text-ink"
              id="save-signing-secret"
            >
              Save this signing secret
            </h3>
            <RevealOnce
              carry="Keep this in your receiver's configuration. Illarin cannot show it again after you dismiss this view."
              copied={copied}
              onCopied={setCopied}
              value={secret}
            />
            <Button onClick={() => setSecret("")} variant="outline">
              I've saved the secret
            </Button>
          </section>
        ) : null}
        <form className="grid gap-5" onSubmit={save}>
          <fieldset className="grid min-w-0 gap-5" disabled={busy}>
            {!current ? <KindChoice kind={kind} onChange={setKind} /> : null}
            <Field
              htmlFor="update-destination-name"
              label="Destination name"
              hint="A name you will recognize when publishing an update."
            >
              <TextInput
                className="bg-plane"
                id="update-destination-name"
                maxLength={48}
                onChange={(event) => setName(event.target.value)}
                required
                value={name}
              />
            </Field>
            {current ? (
              <div className="grid gap-1 text-meta text-mute">
                <span>
                  Stored {kind === "discord" ? "Discord webhook" : "endpoint"}
                </span>
                <code className="break-all text-ink">{current.address}</code>
                {current.channel ? (
                  <span className="wrap-anywhere">
                    Server {current.channel.guildId} · Channel{" "}
                    {current.channel.channelId}
                  </span>
                ) : null}
              </div>
            ) : null}
            <Field
              htmlFor="update-destination-address"
              label={
                current
                  ? "Replacement address"
                  : kind === "discord"
                    ? "Discord webhook address"
                    : "Endpoint address"
              }
              hint={
                current
                  ? "Leave blank to keep the saved address. Replacing a generic endpoint requires verification again."
                  : kind === "discord"
                    ? "Copy the webhook URL from your Discord channel's Integrations settings."
                    : "A public HTTPS endpoint on port 443."
              }
            >
              <TextInput
                autoComplete="off"
                className="bg-plane"
                id="update-destination-address"
                maxLength={300}
                onChange={(event) => setAddress(event.target.value)}
                required={!current}
                spellCheck={false}
                type={kind === "discord" ? "password" : "url"}
                value={address}
              />
            </Field>
            {kind === "discord" ? (
              <StepNote>
                User, role and everyone mentions are disabled. Setup checks the
                channel without posting a message.
              </StepNote>
            ) : (
              <MorphingDisclosure summary="Receiver requirements">
                <div className="mt-3 grid gap-3 text-meta text-mute">
                  <p>
                    Verify the request signature with your signing secret. For{" "}
                    <code className="break-all text-ink">
                      asset.endpoint.verification.v1
                    </code>
                    , return the exact challenge as plain text or a JSON object
                    with a <code className="text-ink">challenge</code> field.
                  </p>
                  <p>
                    Use a public HTTPS address. Redirects, private networks and
                    addresses containing a username or password are refused.
                  </p>
                </div>
              </MorphingDisclosure>
            )}
          </fieldset>
          <Button
            disabled={busy || !name.trim() || (!current && !address.trim())}
            type="submit"
            variant="primary"
          >
            {working === "save"
              ? "Saving…"
              : current
                ? "Save changes"
                : kind === "discord"
                  ? "Connect Discord channel"
                  : "Create webhook"}
          </Button>
        </form>
        {current ? (
          <div className="grid gap-4 border-t border-rule pt-5">
            <output className="text-ui text-ink">
              {current.state === "active"
                ? "Ready for asset updates"
                : current.state === "disabled"
                  ? "Disabled"
                  : "Waiting for verification"}
            </output>
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={busy}
                onClick={() => void verify()}
                variant="outline"
              >
                {working === "verify"
                  ? "Verifying…"
                  : current.state === "disabled"
                    ? "Verify and enable"
                    : "Verify connection"}
              </Button>
              {current.state !== "disabled" ? (
                <Button
                  disabled={busy}
                  onClick={() => void disable()}
                  variant="outline"
                >
                  Disable
                </Button>
              ) : null}
              {kind === "webhook" ? (
                <Button
                  disabled={
                    busy ||
                    Boolean(
                      current.previousSecretUntil &&
                        Date.parse(current.previousSecretUntil) > Date.now(),
                    )
                  }
                  onClick={() => void rotate()}
                  variant="outline"
                >
                  Rotate signing secret
                </Button>
              ) : null}
            </div>
            {current.previousSecretUntil ? (
              <StepNote>
                The previous signing secret expires{" "}
                {new Date(current.previousSecretUntil).toLocaleString()}.
              </StepNote>
            ) : null}
            <Consequence
              action="Remove destination"
              busy={busy}
              confirm="Remove permanently"
              onConfirm={() => void remove()}
            >
              This removes the connection and its saved selections on your
              assets. Reconnecting it requires the address again.
            </Consequence>
          </div>
        ) : null}
      </div>
    </motion.section>
  );
}

function KindChoice({
  kind,
  onChange,
}: {
  kind: AssetUpdateDestination["kind"];
  onChange: (kind: AssetUpdateDestination["kind"]) => void;
}) {
  const [lit, setLit] = useState<string>(kind);
  return (
    <fieldset>
      <legend className="mb-3 text-ui text-ink">Destination type</legend>
      <TravellingHighlight
        chosen={kind}
        className="inline-flex rounded-control bg-plane p-1"
        onLit={setLit}
      >
        {(["discord", "webhook"] as const).map((value) => (
          <button
            aria-pressed={kind === value}
            className={cn(
              "min-h-11 rounded-control px-4 text-ui font-medium outline-offset-3",
              lit === value ? "text-on-accent" : "text-mute",
            )}
            data-cell={value}
            key={value}
            onClick={() => onChange(value)}
            type="button"
          >
            {value === "discord" ? "Discord" : "Generic webhook"}
          </button>
        ))}
      </TravellingHighlight>
    </fieldset>
  );
}
