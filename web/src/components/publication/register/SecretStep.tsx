"use client";

import { KeyRound } from "lucide-react";
import { useState } from "react";
import { StepForm, StepNote } from "@/components/register/StepParts";
import { RevealOnce } from "@/components/ui/reveal-once";
import { rotateDestinationSecret } from "@/lib/api/publication";
import type {
  PublicationDestination,
  RotatedPublicationSecret,
} from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";

export function SecretStep({
  destination,
  onClose,
  onFailure,
  onRotated,
}: {
  destination: PublicationDestination;
  onClose: () => void;
  onFailure: (message: string) => void;
  onRotated: (destination: PublicationDestination) => void;
}) {
  const [turned, setTurned] = useState<RotatedPublicationSecret | null>(null);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);

  async function rotate() {
    setBusy(true);
    const answer = await rotateDestinationSecret(destination.id);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      onClose();
      return;
    }
    onFailure("");
    onRotated(answer.value.destination);
    setTurned(answer.value);
  }

  if (turned) {
    return (
      <TakeTheSecret
        carry="Every request now carries a signature from each secret, separated by a space, in the webhook-signature header. A receiver that accepts either one keeps working through the change."
        copied={copied}
        heading="Copy the new signing secret now"
        line={`This is the only time Illarin can show it. Both secrets are accepted until ${readableMoment(turned.previousSecretUntil)}, so put this one in place before then.`}
        onCopied={setCopied}
        onDone={onClose}
        value={turned.secret}
      />
    );
  }

  return (
    <StepForm
      busy={busy}
      commit="Draw a new secret"
      onCommit={() => void rotate()}
    >
      <StepNote>
        The old secret keeps producing an accepted signature for a day, then
        Illarin forgets it.
      </StepNote>
      <ol className="flex list-none flex-col gap-3">
        {[
          "Illarin draws the new secret and shows it once.",
          "For a day, every request carries a signature from both secrets. A receiver checking either one keeps working.",
          "After that, Illarin forgets the old secret and signs with the new one alone. A receiver still checking the old one stops accepting requests.",
        ].map((step, index) => (
          <li className="flex gap-3" key={step}>
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-accent-wash font-prose text-label text-accent tabular-nums">
              {index + 1}
            </span>
            <span className="font-prose text-meta text-mute">{step}</span>
          </li>
        ))}
      </ol>
      <StepNote>
        The current secret was drawn {readableMoment(destination.secretSetAt)}.
      </StepNote>
    </StepForm>
  );
}

export function TakeTheSecret({
  carry,
  copied,
  heading,
  line,
  onCopied,
  onDone,
  value,
}: {
  carry: string;
  copied: boolean;
  heading: string;
  line: string;
  onCopied: (copied: boolean) => void;
  onDone: () => void;
  value: string;
}) {
  return (
    <div className="flex flex-col gap-5">
      <div>
        <h3 className="flex items-center gap-2 font-display text-ui font-medium text-ink">
          <KeyRound aria-hidden="true" className="size-4 text-accent" />
          {heading}
        </h3>
        <p className="mt-2 max-w-[52ch] font-prose text-meta text-mute">
          {line}
        </p>
      </div>
      <RevealOnce
        carry={carry}
        copied={copied}
        onCopied={onCopied}
        value={value}
      />
      <button
        className="inline-flex min-h-11 items-center justify-center self-start rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90"
        onClick={onDone}
        type="button"
      >
        {copied ? "I have it" : "Close without copying"}
      </button>
    </div>
  );
}
