"use client";

import { useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import { RevealOnce } from "@/components/console/RevealOnce";
import { rotateDestinationSecret } from "@/lib/api/publication";
import type {
  PublicationDestination,
  RotatedPublicationSecret,
} from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import styles from "./RotateSecret.module.css";

/** A new signing secret, and the window the old one keeps working in. */
export function RotateSecret({
  destination,
  onClose,
  onRotated,
  onFailure,
}: {
  destination: PublicationDestination;
  onClose: () => void;
  onRotated: (destination: PublicationDestination) => void;
  onFailure: (message: string) => void;
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
      <FormDialog
        open
        acknowledge
        title="Copy the new signing secret now"
        hint={`This is the only time Illarin can show it. Both secrets are accepted until ${readableMoment(turned.previousSecretUntil)}, so put this one in place before then.`}
        commit={copied ? "I have it" : "Close without copying"}
        onClose={onClose}
        onCommit={onClose}
      >
        <RevealOnce
          carry="Every request now carries a signature from each secret, separated by a space, in the webhook-signature header. A receiver that accepts either one keeps working through the change."
          copied={copied}
          onCopied={setCopied}
          value={turned.secret}
        />
      </FormDialog>
    );
  }

  return (
    <FormDialog
      open
      busy={busy}
      commit="Draw a new secret"
      hint="The old secret keeps producing an accepted signature for a day, then Illarin forgets it."
      onClose={onClose}
      onCommit={() => void rotate()}
      title={`Give ${destination.name} a new signing secret?`}
    >
      <ol className={styles.order}>
        <li>Illarin draws the new secret and shows it once.</li>
        <li>
          For a day, every request carries a signature from both secrets. A
          receiver checking either one keeps working.
        </li>
        <li>
          After that, Illarin forgets the old secret and signs with the new one
          alone. A receiver still checking the old one stops accepting requests.
        </li>
      </ol>
      <p className={styles.since}>
        The current secret was drawn {readableMoment(destination.secretSetAt)}.
      </p>
    </FormDialog>
  );
}
