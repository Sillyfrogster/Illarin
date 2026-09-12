"use client";

import { useState } from "react";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  Row,
  Rows,
} from "@/components/register/RowParts";
import { Consequence, StepForm } from "@/components/register/StepParts";
import { Field, TextInput } from "@/components/ui/field";
import { Sortable, SortableItemHandle } from "@/components/ui/sortable";
import { orderCategories, updateCategory } from "@/lib/api/publication";
import type { PublicationCategory } from "@/lib/api/query";
import { nothingIn } from "@/lib/publication-register";
import { moved } from "@/lib/reorder";

export function CategoryRows({
  categories,
  onFailure,
  onOpen,
  onOrdered,
  onSaved,
}: {
  categories: PublicationCategory[];
  onFailure: (message: string) => void;
  onOpen: (category: PublicationCategory) => void;
  onOrdered: (categories: PublicationCategory[]) => void;
  onSaved: (saved: PublicationCategory) => void;
}) {
  const current = categories.filter((one) => !one.retired);
  const retired = categories.filter((one) => one.retired);

  const [placing, setPlacing] = useState<PublicationCategory[] | null>(null);
  const shown = placing ?? current;

  async function reorder(from: number, to: number) {
    const next = moved(current, from, to);
    setPlacing(next);
    const answer = await orderCategories(
      next.map((one) => one.id).concat(retired.map((one) => one.id)),
    );
    setPlacing(null);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onOrdered(answer.value.categories);
  }

  async function bringBack(category: PublicationCategory) {
    const answer = await updateCategory(category.id, { retired: false });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onSaved(answer.value);
  }

  return (
    <>
      <PanelHead id="register-heading" title="Categories" />

      {current.length === 0 ? (
        <Nothing>{nothingIn("categories", { apps: [] })}</Nothing>
      ) : (
        <Sortable
          disabled={placing !== null}
          ids={shown.map((one) => one.id)}
          labels={(id) =>
            shown.find((one) => one.id === id)?.label ?? "category"
          }
          onMove={(from, to) => void reorder(from, to)}
        >
          <Rows>
            {shown.map((category) => (
              <Row
                aside={
                  <SortableItemHandle
                    disabled={placing !== null || shown.length < 2}
                    label={`Move ${category.label}`}
                  />
                }
                facts={
                  <span className="font-mono text-label">{category.slug}</span>
                }
                key={category.id}
                onOpen={() => onOpen(category)}
                open={`Rename ${category.label}`}
                sortableId={category.id}
                title={category.label}
              />
            ))}
          </Rows>
        </Sortable>
      )}

      {retired.length > 0 ? (
        <Past summary={`${retired.length} retired`}>
          {retired.map((category) => (
            <PastRow
              action={
                <button
                  className="inline-flex min-h-11 items-center rounded-control px-3 font-ui text-meta font-medium text-accent outline-offset-3 hover:underline"
                  onClick={() => bringBack(category)}
                  type="button"
                >
                  Reactivate category
                </button>
              }
              key={category.id}
            >
              {category.label}
            </PastRow>
          ))}
        </Past>
      ) : null}
    </>
  );
}

export function CategoryStep({
  category,
  onClose,
  onFailure,
  onSaved,
}: {
  category: PublicationCategory;
  onClose: () => void;
  onFailure: (message: string) => void;
  onSaved: (saved: PublicationCategory) => void;
}) {
  const [label, setLabel] = useState(category.label);
  const [busy, setBusy] = useState(false);

  async function write(change: { label?: string; retired?: boolean }) {
    setBusy(true);
    const answer = await updateCategory(category.id, change);
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSaved(answer.value);
    onClose();
  }

  return (
    <StepForm
      busy={busy}
      commit="Save"
      onCommit={() => write({ label: label.trim() })}
      ready={Boolean(label.trim())}
      under={
        <Consequence
          action="Retire"
          busy={busy}
          confirm="Retire category"
          onConfirm={() => write({ retired: true })}
        >
          No new post can be filed as {category.label}. Posts already filed as
          it stay where they are, and readers keep the address.
        </Consequence>
      }
    >
      <Field
        hint={`Readers see the new name. Posts and approvals keep referencing ${category.slug}.`}
        htmlFor="category-label"
        label="Name"
      >
        <TextInput
          id="category-label"
          maxLength={48}
          onChange={(event) => setLabel(event.target.value)}
          value={label}
        />
      </Field>
    </StepForm>
  );
}
