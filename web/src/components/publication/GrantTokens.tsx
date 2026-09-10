"use client";

import { Plus } from "lucide-react";
import { useState } from "react";
import { RevealOnce } from "@/components/console/RevealOnce";
import { Field, TextInput, Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { issueToken } from "@/lib/api/publication";
import type { IssuedPublicationToken, PublicationGrant } from "@/lib/api/query";
import { TokenRows } from "./TokenRows";
import { useGrantTokens } from "./use-grant-tokens";

/** An approval's API tokens; the holder makes them, and anyone shown them can revoke. */
export function GrantTokens({
  grant,
  mine = true,
}: {
  grant: PublicationGrant;
  /** Whether this is the holder looking at their own tokens rather than the authority. */
  mine?: boolean;
}) {
  const [failure, setFailure] = useState("");
  const [making, setMaking] = useState(false);
  const { live, reread, spent, tokens } = useGrantTokens(grant.id, setFailure);

  return (
    <section className="flex min-w-0 flex-col gap-4">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-2">
        <h3 className="font-display text-ui font-medium text-ink">
          {mine ? "Tokens for the publication API" : "Their tokens"}
          {tokens ? (
            <span className="ml-2 font-prose text-meta text-mute tabular-nums">
              {live.length}
            </span>
          ) : null}
        </h3>
        {mine && !making ? (
          <button
            className="inline-flex min-h-11 shrink-0 items-center gap-2 rounded-control bg-deep px-4 font-ui text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45"
            onClick={() => setMaking(true)}
            type="button"
          >
            <Plus aria-hidden="true" className="size-4" strokeWidth={2} />
            New token
          </button>
        ) : null}
      </div>

      {failure ? <Trouble>{failure}</Trouble> : null}

      {making ? (
        <NewToken
          appName={grant.app.name}
          grantId={grant.id}
          onDone={() => {
            setMaking(false);
            reread();
          }}
          onFailure={setFailure}
          onIssued={reread}
        />
      ) : null}

      {tokens === null ? (
        <p aria-live="polite" className="font-prose text-meta text-mute">
          Reading {mine ? "your" : "their"} tokens…
        </p>
      ) : live.length === 0 ? (
        <p className="max-w-[52ch] font-prose text-meta text-mute">
          {mine
            ? `Nothing carries your approval yet. A token lets a script or an editor of your own publish for ${grant.app.name} without your password.`
            : "Nothing of theirs can reach the publication API. Only they can make a token; you can stop any of them."}
        </p>
      ) : (
        <TokenRows
          onFailure={setFailure}
          onRevoked={reread}
          revocable
          tokens={live}
        />
      )}

      {spent.length > 0 ? (
        mine ? (
          <MorphingDisclosure
            className="rounded-plate bg-deep px-4 py-3"
            summary={`${spent.length} no longer works`}
          >
            <div className="mt-2">
              <TokenRows
                onFailure={setFailure}
                onRevoked={reread}
                revocable={false}
                tokens={spent}
              />
            </div>
          </MorphingDisclosure>
        ) : (
          <p className="font-prose text-meta text-mute">
            {spent.length === 1
              ? "One more has already stopped working."
              : `${spent.length} more have already stopped working.`}
          </p>
        )
      ) : null}
    </section>
  );
}

/** The two questions a token asks, and the one time its value can be taken. */
function NewToken({
  appName,
  grantId,
  onDone,
  onFailure,
  onIssued,
}: {
  appName: string;
  grantId: string;
  onDone: () => void;
  onFailure: (message: string) => void;
  onIssued: () => void;
}) {
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState("");
  const [made, setMade] = useState<IssuedPublicationToken | null>(null);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);

  async function issue() {
    setBusy(true);
    const written = await issueToken(grantId, {
      expiresAt: expiry ? endOfDay(expiry) : undefined,
      name: name.trim(),
    });
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    setMade(written.value);
    onIssued();
  }

  if (made) {
    return (
      <div className="flex flex-col gap-4 rounded-plate bg-plane p-4 inset-ring-2 inset-ring-accent">
        <div>
          <h4 className="font-display text-ui font-medium text-ink">
            Copy it now
          </h4>
          <p className="mt-1 max-w-[52ch] font-prose text-meta text-mute">
            This is the only time Illarin can show you this token. Nothing here
            can read it back, so if it gets away from you, revoke it and make
            another.
          </p>
        </div>
        <RevealOnce
          carry="Send it as a bearer credential on the publication API. Keep it out of anything you commit or share."
          copied={copied}
          onCopied={setCopied}
          value={made.value}
        />
        <button
          className="inline-flex min-h-11 items-center justify-center self-start rounded-control bg-deep px-5 font-ui text-ui font-medium text-ink outline-offset-3 hover:bg-rule/45"
          onClick={onDone}
          type="button"
        >
          {copied ? "I have it" : "Close without copying"}
        </button>
      </div>
    );
  }

  return (
    <form
      className="flex flex-col gap-5 rounded-plate bg-deep p-4"
      onSubmit={(event) => {
        event.preventDefault();
        if (name.trim() && !busy) void issue();
      }}
    >
      <p className="max-w-[52ch] font-prose text-meta text-mute">
        One token for one tool. It publishes for {appName} exactly as you can,
        and reaches nothing else on Illarin.
      </p>
      <Field
        hint="Name the tool or machine that will carry it, so you know which one to revoke later."
        htmlFor="token-name"
        label="What is it for"
      >
        <TextInput
          autoComplete="off"
          className="bg-plane"
          id="token-name"
          maxLength={48}
          onChange={(event) => setName(event.target.value)}
          placeholder="Release robot"
          value={name}
        />
      </Field>
      <Field
        className="max-w-[16rem]"
        hint="Leave this empty and it works until you revoke it."
        htmlFor="token-expiry"
        label="Stops working on"
      >
        <TextInput
          className="bg-plane"
          id="token-expiry"
          min={tomorrow()}
          onChange={(event) => setExpiry(event.target.value)}
          type="date"
          value={expiry}
        />
      </Field>
      <div className="flex flex-wrap gap-2">
        <button
          className="inline-flex min-h-11 items-center justify-center rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90 disabled:opacity-40"
          disabled={!name.trim() || busy}
          type="submit"
        >
          {busy ? "Making…" : "Make the token"}
        </button>
        <button
          className="inline-flex min-h-11 items-center justify-center rounded-control px-4 font-ui text-ui font-medium text-mute outline-offset-3 hover:text-ink"
          onClick={onDone}
          type="button"
        >
          Cancel
        </button>
      </div>
    </form>
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
