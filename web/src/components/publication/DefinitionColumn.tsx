"use client";

import {
  ArrowDown,
  ArrowUp,
  Award,
  ImageUp,
  Plus,
  RotateCcw,
  X,
} from "lucide-react";
import Image from "next/image";
import { type FormEvent, useRef, useState } from "react";
import {
  defineDistinction,
  orderDistinctions,
  updateDistinction,
  uploadMark,
} from "@/lib/api/distinctions";
import type { Distinction, DistinctionForm } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./DefinitionColumn.module.css";

export function DefinitionColumn({
  form,
  heading,
  hint,
  definitions,
  onChanged,
  onFailure,
}: {
  form: DistinctionForm;
  heading: string;
  hint: string;
  definitions: Distinction[];
  onChanged: (definitions: Distinction[]) => void;
  onFailure: (message: string) => void;
}) {
  const [name, setName] = useState("");
  const [explanation, setExplanation] = useState("");
  const [busy, setBusy] = useState(false);
  const [marking, setMarking] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  const mine = definitions.filter((definition) => definition.form === form);
  const current = mine.filter((definition) => !definition.retired);
  const retired = mine.filter((definition) => definition.retired);

  async function add(event: FormEvent) {
    event.preventDefault();
    if (!name.trim() || busy) return;
    setBusy(true);
    const answer = await defineDistinction(
      form,
      name.trim(),
      explanation.trim(),
    );
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    setName("");
    setExplanation("");
    onChanged([...definitions, answer.value]);
  }

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (target < 0 || target >= current.length) return;
    const answer = await orderDistinctions(
      form,
      moved(
        mine.map((definition) => definition.id),
        mine.indexOf(current[index]),
        mine.indexOf(current[target]),
      ),
    );
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(answer.value.definitions);
  }

  async function setRetired(definition: Distinction, retired: boolean) {
    const answer = await updateDistinction(definition.id, { retired });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(
      definitions.map((one) =>
        one.id === definition.id ? (answer.value as Distinction) : one,
      ),
    );
  }

  function chooseMark(definition: Distinction) {
    setMarking(definition.id);
    fileInput.current?.click();
  }

  async function sendMark(file: File | undefined) {
    if (!file || !marking) return;
    const answer = await uploadMark(marking, file);
    setMarking("");
    if (fileInput.current) fileInput.current.value = "";
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(
      definitions.map((one) =>
        one.id === marking ? (answer.value as Distinction) : one,
      ),
    );
  }

  return (
    <section className={styles.column}>
      <header className={styles.heading}>
        <h3>{heading}</h3>
        <p>{hint}</p>
      </header>

      {current.length > 0 ? (
        <ol className={styles.list}>
          {current.map((definition, index) => (
            <li className={styles.row} key={definition.id}>
              {form === "badge" ? (
                <span className={styles.mark}>
                  {definition.mark ? (
                    <Image
                      src={definition.mark.url}
                      alt=""
                      width={28}
                      height={28}
                      unoptimized
                    />
                  ) : (
                    <Award size={16} strokeWidth={1.7} aria-hidden="true" />
                  )}
                </span>
              ) : null}
              <span className={styles.name}>
                {definition.name}
                {definition.explanation ? (
                  <span className={styles.explanation}>
                    {definition.explanation}
                  </span>
                ) : null}
              </span>
              <span className={styles.rowActions}>
                <button
                  type="button"
                  onClick={() => reorder(index, -1)}
                  disabled={index === 0}
                  aria-label={`Move ${definition.name} up in ${heading}`}
                >
                  <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => reorder(index, 1)}
                  disabled={index === current.length - 1}
                  aria-label={`Move ${definition.name} down in ${heading}`}
                >
                  <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                {form === "badge" ? (
                  <button
                    type="button"
                    onClick={() => chooseMark(definition)}
                    aria-label={`Upload a mark for ${definition.name}`}
                  >
                    <ImageUp size={15} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                ) : null}
                <button
                  type="button"
                  onClick={() => setRetired(definition, true)}
                  aria-label={`Retire ${definition.name}`}
                >
                  <X size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
              </span>
            </li>
          ))}
        </ol>
      ) : (
        <p className={styles.empty}>Nothing defined yet.</p>
      )}

      <form className={styles.add} onSubmit={add}>
        <label className="sr-only" htmlFor={`${form}-name`}>
          {heading} name
        </label>
        <input
          id={`${form}-name`}
          value={name}
          maxLength={48}
          placeholder="Name"
          onChange={(event) => setName(event.target.value)}
        />
        {form === "badge" ? (
          <>
            <label className="sr-only" htmlFor={`${form}-explanation`}>
              What this badge is for
            </label>
            <input
              id={`${form}-explanation`}
              value={explanation}
              maxLength={200}
              placeholder="What it is for"
              onChange={(event) => setExplanation(event.target.value)}
            />
          </>
        ) : null}
        <button type="submit" disabled={busy || !name.trim()}>
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          Add
        </button>
      </form>

      {retired.length > 0 ? (
        <div className={styles.retired}>
          <h4>Retired</h4>
          <ul>
            {retired.map((definition) => (
              <li key={definition.id}>
                {definition.name}
                <button
                  type="button"
                  onClick={() => setRetired(definition, false)}
                  aria-label={`Bring ${definition.name} back`}
                >
                  <RotateCcw size={14} strokeWidth={1.8} aria-hidden="true" />
                  Bring back
                </button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {form === "badge" ? (
        <input
          className="sr-only"
          ref={fileInput}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          onChange={(event) => sendMark(event.target.files?.[0])}
        />
      ) : null}
    </section>
  );
}
