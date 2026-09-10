"use client";

import { ArrowDown, ArrowUp } from "lucide-react";
import { useState } from "react";
import { Field, TextInput } from "@/components/ui/field";
import { orderCategories, updateCategory } from "@/lib/api/publication";
import type { PublicationCategory } from "@/lib/api/query";
import { nothingIn } from "@/lib/publication-register";
import { moved } from "@/lib/reorder";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  Row,
  RowMove,
  Rows,
} from "./RowParts";
import { Consequence, StepForm } from "./StepParts";

/** What a post can be filed as, in the order readers meet on the blog. */
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

  async function reorder(index: number, step: number) {
    const target = index + step;
    if (target < 0 || target >= current.length) return;
    const answer = await orderCategories(
      moved(
        categories.map((one) => one.id),
        categories.indexOf(current[index]),
        categories.indexOf(current[target]),
      ),
    );
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
        <Rows>
          {current.map((category, index) => (
            <Row
              aside={
                <>
                  <RowMove
                    disabled={index === 0}
                    icon={ArrowUp}
                    label={`Move ${category.label} up`}
                    onClick={() => reorder(index, -1)}
                  />
                  <RowMove
                    disabled={index === current.length - 1}
                    icon={ArrowDown}
                    label={`Move ${category.label} down`}
                    onClick={() => reorder(index, 1)}
                  />
                </>
              }
              facts={
                <span className="font-mono text-label">{category.slug}</span>
              }
              key={category.id}
              onOpen={() => onOpen(category)}
              open={`Rename ${category.label}`}
              title={category.label}
            />
          ))}
        </Rows>
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
                  Bring back
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

/** Renaming a category, or taking it out of what a post can be filed as. */
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
          confirm="Retire it"
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
