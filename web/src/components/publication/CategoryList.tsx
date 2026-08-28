"use client";

import { ArrowDown, ArrowUp } from "lucide-react";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { Section } from "@/components/console/Section";
import { orderCategories, updateCategory } from "@/lib/api/publication";
import type { PublicationCategory } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./AppDialog.module.css";

export function CategoryList({
  categories,
  onChanged,
  onFailure,
}: {
  categories: PublicationCategory[];
  onChanged: (categories: PublicationCategory[]) => void;
  onFailure: (message: string) => void;
}) {
  const [editing, setEditing] = useState<PublicationCategory | null>(null);

  const current = categories.filter((category) => !category.retired);
  const retired = categories.filter((category) => category.retired);

  function replace(saved: PublicationCategory) {
    onChanged(categories.map((one) => (one.id === saved.id ? saved : one)));
  }

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (target < 0 || target >= current.length) return;
    const answer = await orderCategories(
      moved(
        categories.map((category) => category.id),
        categories.indexOf(current[index]),
        categories.indexOf(current[target]),
      ),
    );
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(answer.value.categories);
  }

  async function setRetired(category: PublicationCategory, retired: boolean) {
    const answer = await updateCategory(category.id, { retired });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    replace(answer.value);
  }

  return (
    <Section
      title="Categories"
      count={current.length}
      retiredLabel={
        retired.length > 0 ? `${retired.length} retired` : undefined
      }
      retired={
        retired.length > 0 ? (
          <ul className={rows.pastList}>
            {retired.map((category) => (
              <li className={rows.pastRow} key={category.id}>
                <span>{category.label}</span>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setRetired(category, false)}
                >
                  Bring back
                </button>
              </li>
            ))}
          </ul>
        ) : null
      }
    >
      <ol className={rows.list}>
        {current.map((category, index) => (
          <li className={rows.row} data-plain="true" key={category.id}>
            <span className={rows.name}>
              {category.label}{" "}
              <span className={rows.slug}>{category.slug}</span>
            </span>
            <span className={rows.detail} />
            <span className={rows.actions}>
              <button
                type="button"
                className={rows.iconButton}
                onClick={() => reorder(index, -1)}
                disabled={index === 0}
                aria-label={`Move ${category.label} up`}
              >
                <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
              </button>
              <button
                type="button"
                className={rows.iconButton}
                onClick={() => reorder(index, 1)}
                disabled={index === current.length - 1}
                aria-label={`Move ${category.label} down`}
              >
                <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
              </button>
              <button
                type="button"
                className={rows.textButton}
                onClick={() => setEditing(category)}
              >
                Edit
              </button>
            </span>
          </li>
        ))}
      </ol>

      {editing ? (
        <CategoryDialog
          key={editing.id}
          category={editing}
          onClose={() => setEditing(null)}
          onSaved={replace}
          onFailure={onFailure}
          onRetire={() => setRetired(editing, true)}
        />
      ) : null}
    </Section>
  );
}

function CategoryDialog({
  category,
  onClose,
  onSaved,
  onFailure,
  onRetire,
}: {
  category: PublicationCategory;
  onClose: () => void;
  onSaved: (saved: PublicationCategory) => void;
  onFailure: (message: string) => void;
  onRetire: () => void;
}) {
  const [label, setLabel] = useState(category.label);
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    const answer = await updateCategory(category.id, { label: label.trim() });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSaved(answer.value);
    onClose();
  }

  return (
    <FormDialog
      open
      title="Rename this category"
      hint={`Readers see the new name. Posts and approvals keep referencing ${category.slug}.`}
      commit="Save"
      busy={busy}
      ready={Boolean(label.trim())}
      onClose={onClose}
      onCommit={save}
      destructive={
        <button
          type="button"
          className={styles.retire}
          onClick={() => {
            onRetire();
            onClose();
          }}
        >
          Retire
        </button>
      }
    >
      <Field label="Name" htmlFor="category-label">
        <input
          id="category-label"
          value={label}
          maxLength={48}
          onChange={(event) => setLabel(event.target.value)}
        />
      </Field>
    </FormDialog>
  );
}
