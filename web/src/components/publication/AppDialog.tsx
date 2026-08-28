"use client";

import { ImageUp, Package } from "lucide-react";
import Image from "next/image";
import { useRef, useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { configureApp, updateApp, uploadAppMark } from "@/lib/api/publication";
import type { PublicationApp } from "@/lib/api/query";
import styles from "./AppDialog.module.css";

export function AppDialog({
  existing,
  onClose,
  onSaved,
  onFailure,
}: {
  existing: PublicationApp | null;
  onClose: () => void;
  onSaved: (saved: PublicationApp, added: boolean) => void;
  onFailure: (message: string) => void;
}) {
  const [name, setName] = useState(existing?.name ?? "");
  const [slug, setSlug] = useState(existing?.slug ?? "");
  const [home, setHome] = useState(existing?.home ?? "");
  const [chosen, setChosen] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [busy, setBusy] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const ready = Boolean(name.trim() && slug.trim() && home.trim());

  function take(file: File | undefined) {
    if (!file) return;
    setChosen(file);
    setPreview(URL.createObjectURL(file));
  }

  async function save() {
    setBusy(true);
    const written = existing
      ? await updateApp(existing.id, {
          name: name.trim(),
          slug: slug.trim(),
          home: home.trim(),
        })
      : await configureApp(slug.trim(), name.trim(), home.trim());
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    let saved = written.value;
    if (chosen) {
      const marked = await uploadAppMark(saved.id, chosen);
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

  async function retire() {
    if (!existing) return;
    setBusy(true);
    const answer = await updateApp(existing.id, { retired: true });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSaved(answer.value, false);
    onClose();
  }

  const shown = preview || existing?.mark?.url || "";

  return (
    <FormDialog
      open
      title={existing ? "Edit this app" : "Add an app"}
      hint="A project Illarin publishes official updates for. Adding one approves nobody."
      commit={existing ? "Save" : "Add"}
      busy={busy}
      ready={ready}
      onClose={onClose}
      onCommit={save}
      destructive={
        existing ? (
          <button type="button" className={styles.retire} onClick={retire}>
            Retire
          </button>
        ) : null
      }
    >
      <Field label="Name" htmlFor="app-name">
        <input
          id="app-name"
          value={name}
          maxLength={48}
          placeholder="Lumiverse"
          onChange={(event) => setName(event.target.value)}
        />
      </Field>

      <Field
        label="Slug"
        htmlFor="app-slug"
        hint="What a post references. Renaming the app leaves it alone."
      >
        <input
          id="app-slug"
          value={slug}
          maxLength={40}
          placeholder="lumiverse"
          onChange={(event) => setSlug(event.target.value)}
        />
      </Field>

      <Field label="Address" htmlFor="app-home">
        <input
          id="app-home"
          type="url"
          value={home}
          maxLength={300}
          placeholder="https://lumiverse.app"
          onChange={(event) => setHome(event.target.value)}
        />
      </Field>

      <Field label="Mark">
        <div className={styles.mark}>
          <span className={styles.preview} data-blank={!shown || undefined}>
            {shown ? (
              <Image src={shown} alt="" width={48} height={48} unoptimized />
            ) : (
              <Package size={19} strokeWidth={1.6} aria-hidden="true" />
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
