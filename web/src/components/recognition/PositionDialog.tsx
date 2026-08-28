"use client";

import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { defineDistinction, updateDistinction } from "@/lib/api/distinctions";
import type { Distinction } from "@/lib/api/query";
import styles from "./RecognitionDialog.module.css";

const NAME_LIMIT = 48;

export function PositionDialog({
  open,
  existing,
  onClose,
  onSaved,
  onFailure,
}: {
  open: boolean;
  existing: Distinction | null;
  onClose: () => void;
  onSaved: (saved: Distinction, added: boolean) => void;
  onFailure: (message: string) => void;
}) {
  const [name, setName] = useState(existing?.name ?? "");
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    const written = existing
      ? await updateDistinction(existing.id, { name: name.trim() })
      : await defineDistinction("position", name.trim(), "");
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    onSaved(written.value, !existing);
    onClose();
  }

  async function setRetired(retired: boolean) {
    if (!existing) return;
    setBusy(true);
    const answer = await updateDistinction(existing.id, { retired });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSaved(answer.value, false);
    onClose();
  }

  return (
    <FormDialog
      open={open}
      title={existing ? "Rename this position" : "Add a position"}
      hint="A job somebody does at Illarin. Every position an account holds shows on their profile."
      commit={existing ? "Save" : "Add"}
      busy={busy}
      ready={Boolean(name.trim())}
      onClose={onClose}
      onCommit={save}
      destructive={
        existing ? (
          <button
            type="button"
            className={styles.retire}
            onClick={() => setRetired(!existing.retired)}
          >
            {existing.retired ? "Bring back" : "Retire"}
          </button>
        ) : null
      }
    >
      <Field label="Name" htmlFor="position-name">
        <input
          id="position-name"
          value={name}
          maxLength={NAME_LIMIT}
          placeholder="Developer"
          onChange={(event) => setName(event.target.value)}
        />
      </Field>
    </FormDialog>
  );
}
