"use client";

import { ArrowDown, ArrowUp, Check, RotateCcw, X } from "lucide-react";
import { type FormEvent, useCallback, useEffect, useState } from "react";
import {
  orderCategories,
  readCategories,
  updateCategory,
} from "@/lib/api/publication";
import type { PublicationCategory } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./PublicationSection.module.css";

export function PublicationCategories() {
  const [categories, setCategories] = useState<PublicationCategory[] | null>(
    null,
  );
  const [failure, setFailure] = useState("");
  const [editing, setEditing] = useState("");
  const [label, setLabel] = useState("");

  const load = useCallback(async () => {
    const answer = await readCategories();
    if (answer.error) {
      setFailure(answer.error);
      return;
    }
    setFailure("");
    setCategories(answer.value?.categories ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const current = (categories ?? []).filter((category) => !category.retired);
  const retired = (categories ?? []).filter((category) => category.retired);

  function replace(changed: PublicationCategory) {
    setCategories((known) =>
      (known ?? []).map((one) => (one.id === changed.id ? changed : one)),
    );
  }

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (!categories || target < 0 || target >= current.length) return;
    const answer = await orderCategories(
      moved(
        categories.map((category) => category.id),
        categories.indexOf(current[index]),
        categories.indexOf(current[target]),
      ),
    );
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setCategories(answer.value.categories);
  }

  async function rename(event: FormEvent) {
    event.preventDefault();
    if (!editing || !label.trim()) return;
    const answer = await updateCategory(editing, { label: label.trim() });
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setEditing("");
    replace(answer.value);
  }

  async function setRetired(
    category: PublicationCategory,
    retiredNow: boolean,
  ) {
    const answer = await updateCategory(category.id, { retired: retiredNow });
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    replace(answer.value);
  }

  if (!categories) {
    return (
      <p className={styles.loading} aria-live="polite">
        {failure || "Reading the categories…"}
      </p>
    );
  }

  return (
    <div className={styles.region}>
      {failure ? (
        <p className={styles.failure} role="alert">
          {failure}
        </p>
      ) : null}

      <ol className={styles.list}>
        {current.map((category, index) => (
          <li className={styles.row} key={category.id}>
            {editing === category.id ? (
              <form className={styles.rename} onSubmit={rename}>
                <label className="sr-only" htmlFor={`label-${category.id}`}>
                  What {category.slug} is called
                </label>
                <input
                  id={`label-${category.id}`}
                  value={label}
                  maxLength={48}
                  onChange={(event) => setLabel(event.target.value)}
                  /* biome-ignore lint/a11y/noAutofocus: the row became this field */
                  autoFocus
                />
                <button type="submit" disabled={!label.trim()}>
                  <Check size={15} strokeWidth={2} aria-hidden="true" />
                  Save
                </button>
                <button type="button" onClick={() => setEditing("")}>
                  Cancel
                </button>
              </form>
            ) : (
              <>
                <span className={styles.identity}>
                  <span className={styles.name}>{category.label}</span>
                  <span className={styles.slug}>{category.slug}</span>
                </span>
                <button
                  type="button"
                  className={styles.textAction}
                  onClick={() => {
                    setEditing(category.id);
                    setLabel(category.label);
                  }}
                >
                  Rename
                </button>
                <span className={styles.rowActions}>
                  <button
                    type="button"
                    onClick={() => reorder(index, -1)}
                    disabled={index === 0}
                    aria-label={`Move ${category.label} up`}
                  >
                    <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                  <button
                    type="button"
                    onClick={() => reorder(index, 1)}
                    disabled={index === current.length - 1}
                    aria-label={`Move ${category.label} down`}
                  >
                    <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                  <button
                    type="button"
                    onClick={() => setRetired(category, true)}
                    aria-label={`Retire ${category.label}`}
                  >
                    <X size={15} strokeWidth={1.8} aria-hidden="true" />
                  </button>
                </span>
              </>
            )}
          </li>
        ))}
      </ol>

      <p className={styles.note}>
        The slug beside each name is what posts and approvals reference. It does
        not change when you rename a category.
      </p>

      {retired.length > 0 ? (
        <div className={styles.retired}>
          <h3>Retired</h3>
          <ul>
            {retired.map((category) => (
              <li key={category.id}>
                {category.label}
                <button
                  type="button"
                  onClick={() => setRetired(category, false)}
                  aria-label={`Bring ${category.label} back`}
                >
                  <RotateCcw size={14} strokeWidth={1.8} aria-hidden="true" />
                  Bring back
                </button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
