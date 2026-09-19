"use client";

import { KeyRound, Power, ShieldCheck } from "lucide-react";
import { type FormEvent, useState } from "react";
import {
  Consequence,
  StepAction,
  StepNote,
} from "@/components/register/StepParts";
import { Button } from "@/components/ui/button";
import { Field, Said, TextInput, Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { RevealOnce } from "@/components/ui/reveal-once";
import { TravellingHighlight } from "@/components/ui/travelling-highlight";
import {
  addIntegration,
  changeIntegration,
  disableIntegration,
  removeIntegration,
  rotateIntegrationSecret,
  verifyIntegration,
  type WorkIntegration,
} from "@/lib/api/integrations";
import { cn } from "@/lib/cn";
import { integrationRotating, integrationStanding } from "@/lib/integrations";

export function IntegrationEditor({
  existing,
  busy,
  onBusy,
  onSaved,
  onRemoved,
}: {
  existing: WorkIntegration | null;
  busy: boolean;
  onBusy: (busy: boolean) => void;
  onSaved: (integration: WorkIntegration) => void;
  onRemoved: (id: string) => void;
}) {
  const [current, setCurrent] = useState(existing);
  const [integrationType, setIntegrationType] = useState<
    WorkIntegration["type"]
  >(existing?.type ?? "discord");
  const [name, setName] = useState(existing?.name ?? "");
  const [address, setAddress] = useState("");
  const [secret, setSecret] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [working, setWorking] = useState("");

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

  function receive(saved: WorkIntegration) {
    setCurrent(saved);
    onSaved(saved);
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    start("save");
    const answer = current
      ? await changeIntegration(current.id, {
          name: name.trim(),
          address: address.trim() || undefined,
        })
      : await addIntegration({
          name: name.trim(),
          type: integrationType,
          address: address.trim(),
        });
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not save the integration.");
      return;
    }
    setAddress("");
    if ("integration" in answer.value) {
      receive(answer.value.integration);
      setSecret(answer.value.secret ?? "");
      setCopied(false);
    } else receive(answer.value);
    setNotice(
      integrationType === "discord"
        ? "Discord channel saved."
        : "Integration saved.",
    );
  }

  async function verify() {
    if (!current || busy) return;
    start("verify");
    const answer = await verifyIntegration(current.id);
    finish();
    if (!answer.value) {
      setError(
        answer.error ||
          "Verification failed. Check the receiver and try again.",
      );
      return;
    }
    receive(answer.value);
    setNotice("Verified. This integration is ready to choose for your work.");
  }

  async function disable() {
    if (!current || busy) return;
    start("disable");
    const answer = await disableIntegration(current.id);
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not disable the integration.");
      return;
    }
    receive(answer.value);
    setNotice("Integration disabled.");
  }

  async function rotate() {
    if (!current || busy) return;
    start("rotate");
    const answer = await rotateIntegrationSecret(current.id);
    finish();
    if (!answer.value) {
      setError(answer.error || "Could not rotate the signing secret.");
      return;
    }
    receive(answer.value.integration);
    setSecret(answer.value.secret ?? "");
    setCopied(false);
    setNotice(
      "Signing secret rotated. The previous secret remains valid for 24 hours.",
    );
  }

  async function remove() {
    if (!current || busy) return;
    start("remove");
    const answer = await removeIntegration(current.id);
    finish();
    if (answer.error) {
      setError(answer.error);
      return;
    }
    onRemoved(current.id);
  }

  return (
    <div className="grid gap-6">
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
          {!current ? (
            <TypeChoice
              integrationType={integrationType}
              onChange={setIntegrationType}
            />
          ) : null}
          <Field
            htmlFor="integration-name"
            label="Integration name"
            hint="A name you will recognize when publishing a version."
          >
            <TextInput
              id="integration-name"
              maxLength={48}
              onChange={(event) => setName(event.target.value)}
              required
              value={name}
            />
          </Field>
          {current ? (
            <div className="grid gap-1 rounded-control bg-deep px-3.5 py-3 text-meta text-mute">
              <span>
                Saved {integrationType === "discord" ? "channel" : "endpoint"},
                masked
              </span>
              <code className="break-all text-ink">{current.address}</code>
            </div>
          ) : null}
          <Field
            htmlFor="integration-address"
            label={
              current
                ? "Replacement address"
                : integrationType === "discord"
                  ? "Discord webhook address"
                  : "Endpoint address"
            }
            hint={
              current
                ? "Leave blank to keep the saved address. Replacing a generic endpoint requires verification again."
                : integrationType === "discord"
                  ? "Copy the webhook URL from your Discord channel's Integrations settings."
                  : "A public HTTPS endpoint on port 443."
            }
          >
            <TextInput
              autoComplete="off"
              id="integration-address"
              maxLength={300}
              onChange={(event) => setAddress(event.target.value)}
              required={!current}
              spellCheck={false}
              type={integrationType === "discord" ? "password" : "url"}
              value={address}
            />
          </Field>
          {integrationType === "discord" ? (
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
        <button
          className="inline-flex min-h-11 items-center justify-center rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90 disabled:opacity-40"
          disabled={busy || !name.trim() || (!current && !address.trim())}
          type="submit"
        >
          {working === "save"
            ? "Saving…"
            : current
              ? "Save changes"
              : integrationType === "discord"
                ? "Connect this channel"
                : "Create the endpoint"}
        </button>
      </form>
      {current ? (
        <div className="grid gap-4 border-t border-rule/45 pt-5">
          <StepNote>{integrationStanding(current)}</StepNote>
          <div className="flex flex-wrap gap-2">
            <StepAction
              busy={busy}
              icon={ShieldCheck}
              onClick={() => void verify()}
            >
              {working === "verify"
                ? "Verifying…"
                : current.state === "disabled"
                  ? "Verify and switch on"
                  : "Verify again"}
            </StepAction>
            {current.state !== "disabled" ? (
              <StepAction
                busy={busy}
                icon={Power}
                onClick={() => void disable()}
              >
                Switch off
              </StepAction>
            ) : null}
            {current.type === "webhook" ? (
              <StepAction
                busy={busy || integrationRotating(current)}
                icon={KeyRound}
                onClick={() => void rotate()}
              >
                Rotate signing secret
              </StepAction>
            ) : null}
          </div>
          <Consequence
            action="Remove integration"
            busy={busy}
            confirm="Remove permanently"
            onConfirm={() => void remove()}
          >
            This removes the connection and every selection on your work that
            names it. Connecting it again needs the address again.
          </Consequence>
        </div>
      ) : null}
    </div>
  );
}

function TypeChoice({
  integrationType,
  onChange,
}: {
  integrationType: WorkIntegration["type"];
  onChange: (integrationType: WorkIntegration["type"]) => void;
}) {
  const [lit, setLit] = useState<string>(integrationType);
  return (
    <fieldset>
      <legend className="mb-3 text-ui text-ink">Integration type</legend>
      <TravellingHighlight
        chosen={integrationType}
        className="inline-flex rounded-control bg-deep p-1"
        onLit={setLit}
      >
        {(["discord", "webhook"] as const).map((value) => (
          <button
            aria-pressed={integrationType === value}
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
