"use client";

import { ImageUp, Trash2 } from "lucide-react";
import Image from "next/image";
import { useRef, useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import {
  clearMark,
  defineDistinction,
  updateDistinction,
  uploadMark,
} from "@/lib/api/distinctions";
import type { Distinction } from "@/lib/api/query";
import styles from "./RecognitionDialog.module.css";

const NAME_LIMIT = 48;
const EARNED_LIMIT = 200;

export function RecognitionDialog({
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
  const [earned, setEarned] = useState(existing?.explanation ?? "");
  const [mark, setMark] = useState(existing?.mark ?? null);
  const [chosen, setChosen] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [busy, setBusy] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  function take(file: File | undefined) {
    if (!file) return;
    setChosen(file);
    setPreview(URL.createObjectURL(file));
  }

  async function drop() {
    if (chosen) {
      setChosen(null);
      setPreview("");
      if (fileInput.current) fileInput.current.value = "";
      return;
    }
    if (!existing || !mark) return;
    setBusy(true);
    const answer = await clearMark(existing.id);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    setMark(null);
    onSaved(answer.value, false);
  }

  async function save() {
    setBusy(true);
    const written = existing
      ? await updateDistinction(existing.id, {
          name: name.trim(),
          explanation: earned.trim(),
        })
      : await defineDistinction("title", name.trim(), earned.trim());
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    let saved = written.value;
    if (chosen) {
      const marked = await uploadMark(saved.id, chosen);
      if (marked.error || !marked.value) {
        setBusy(false);
        onSaved(saved, !existing);
        onFailure(marked.error ?? "");
        return;
      }
      saved = marked.value;
    }
    setBusy(false);
    onSaved(saved, !existing);
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

  const shown = preview || mark?.url || "";

  return (
    <FormDialog
      open={open}
      title={existing ? "Edit this recognition" : "Add a recognition"}
      hint="Give it a mark and it shows as a badge. Leave the mark off and it shows as a title."
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
      <Field label="Name" htmlFor="recognition-name">
        <input
          id="recognition-name"
          value={name}
          maxLength={NAME_LIMIT}
          placeholder="First light"
          onChange={(event) => setName(event.target.value)}
        />
      </Field>

      <Field
        label="What earns it"
        htmlFor="recognition-earned"
        hint="Written for the person reading a profile. A recognition nobody can explain is one nobody wants."
      >
        <input
          id="recognition-earned"
          value={earned}
          maxLength={EARNED_LIMIT}
          placeholder="Published a first asset"
          onChange={(event) => setEarned(event.target.value)}
        />
      </Field>

      <Field label="Mark">
        <div className={styles.mark}>
          <span className={styles.preview} data-blank={!shown || undefined}>
            {shown ? (
              <Image src={shown} alt="" width={56} height={56} unoptimized />
            ) : (
              "No mark"
            )}
          </span>
          <button
            type="button"
            className={styles.choose}
            onClick={() => fileInput.current?.click()}
          >
            <ImageUp size={15} strokeWidth={1.8} aria-hidden="true" />
            {shown ? "Replace" : "Upload one"}
          </button>
          {shown ? (
            <button type="button" className={styles.remove} onClick={drop}>
              <Trash2 size={15} strokeWidth={1.8} aria-hidden="true" />
              Remove
            </button>
          ) : null}
          <input
            className="sr-only"
            ref={fileInput}
            type="file"
            accept="image/png,image/jpeg,image/webp,image/gif"
            onChange={(event) => take(event.target.files?.[0])}
          />
        </div>
      </Field>
    </FormDialog>
  );
}
