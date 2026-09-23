"use client";

import { Hash } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextInput, Trouble } from "@/components/ui/field";
import {
  connectDiscordChannel,
  type DiscordScope,
  disconnectDiscordChannel,
  readDiscordChannel,
} from "@/lib/api/integrations";

/** DiscordChannel saves, replaces or removes the one Discord channel Illarin posts to. */
export function DiscordChannel({ scope }: { scope: DiscordScope }) {
  const [connected, setConnected] = useState<boolean | null>(null);
  const [address, setAddress] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    void readDiscordChannel(scope, controller.signal).then((answer) => {
      if (controller.signal.aborted) return;
      if (answer.value) setConnected(answer.value.connected);
      else setError(answer.error ?? "Illarin could not read the channel.");
    });
    return () => controller.abort();
  }, [scope]);

  async function run(change: () => ReturnType<typeof readDiscordChannel>) {
    setBusy(true);
    setError("");
    const answer = await change();
    setBusy(false);
    if (!answer.value) {
      setError(answer.error ?? "That did not work. Try again.");
      return;
    }
    setConnected(answer.value.connected);
    setAddress("");
  }

  function save(event: FormEvent) {
    event.preventDefault();
    void run(() => connectDiscordChannel(scope, address.trim()));
  }

  return (
    <div className="grid max-w-[36rem] gap-4">
      <p className="flex items-center gap-2 font-ui text-ui text-ink">
        <Hash
          aria-hidden="true"
          className={connected ? "size-4 text-accent" : "size-4 text-mute"}
        />
        {connected === null
          ? "Checking…"
          : connected
            ? "Connected to a Discord channel"
            : "No Discord channel"}
      </p>
      {error ? <Trouble>{error}</Trouble> : null}
      <form className="grid gap-3" onSubmit={save}>
        <Field
          hint="In Discord: channel settings, Integrations, Webhooks, Copy webhook URL."
          htmlFor={`discord-${scope}`}
          label={connected ? "Replace the webhook address" : "Webhook address"}
        >
          <TextInput
            autoComplete="off"
            id={`discord-${scope}`}
            onChange={(event) => setAddress(event.target.value)}
            placeholder="https://discord.com/api/webhooks/…"
            spellCheck={false}
            type="url"
            value={address}
          />
        </Field>
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={busy || !address.trim()}
            loading={busy}
            type="submit"
            variant="primary"
          >
            {connected ? "Replace channel" : "Connect channel"}
          </Button>
          {connected ? (
            <Button
              disabled={busy}
              onClick={() => void run(() => disconnectDiscordChannel(scope))}
              variant="ghost"
            >
              Remove channel
            </Button>
          ) : null}
        </div>
      </form>
    </div>
  );
}
