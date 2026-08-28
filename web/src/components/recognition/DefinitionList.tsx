"use client";

import { ArrowDown, ArrowUp, Plus } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import { Section } from "@/components/console/Section";
import { orderDistinctions } from "@/lib/api/distinctions";
import type { Distinction, DistinctionForm } from "@/lib/api/query";
import { moved } from "@/lib/reorder";
import styles from "./DefinitionList.module.css";
import { PositionDialog } from "./PositionDialog";
import { RecognitionDialog } from "./RecognitionDialog";

/** One form's definitions, badges before titles so a form's arrows move through one unbroken run of rows. */
export function DefinitionList({
  kind,
  title,
  definitions,
  onChanged,
  onFailure,
}: {
  kind: "position" | "recognition";
  title: string;
  definitions: Distinction[];
  onChanged: (definitions: Distinction[]) => void;
  onFailure: (message: string) => void;
}) {
  const [editing, setEditing] = useState<Distinction | null>(null);
  const [adding, setAdding] = useState(false);

  const forms: DistinctionForm[] =
    kind === "position" ? ["position"] : ["badge", "title"];
  const mine = definitions.filter((one) => forms.includes(one.form));
  const current = mine
    .filter((one) => !one.retired)
    .sort((a, b) => forms.indexOf(a.form) - forms.indexOf(b.form));
  const retired = mine.filter((one) => one.retired);

  function replace(saved: Distinction, added: boolean) {
    onChanged(
      added
        ? [...definitions, saved]
        : definitions.map((one) => (one.id === saved.id ? saved : one)),
    );
    if (editing?.id === saved.id) setEditing(saved);
  }

  async function reorder(one: Distinction, step: number) {
    const peers = current.filter((other) => other.form === one.form);
    const at = peers.indexOf(one);
    const target = at + step;
    if (target < 0 || target >= peers.length) return;
    const family = definitions.filter((other) => other.form === one.form);
    const answer = await orderDistinctions(
      one.form,
      moved(
        family.map((other) => other.id),
        family.indexOf(peers[at]),
        family.indexOf(peers[target]),
      ),
    );
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(answer.value.definitions);
  }

  function movable(one: Distinction, step: number) {
    const peers = current.filter((other) => other.form === one.form);
    const at = peers.indexOf(one);
    return at + step >= 0 && at + step < peers.length;
  }

  const Dialog = kind === "position" ? PositionDialog : RecognitionDialog;

  return (
    <Section
      title={title}
      count={current.length}
      action={
        <button
          type="button"
          className={styles.add}
          onClick={() => setAdding(true)}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          {kind === "position" ? "Add a position" : "Add a recognition"}
        </button>
      }
      retiredLabel={
        retired.length > 0 ? `${retired.length} retired` : undefined
      }
      retired={
        retired.length > 0 ? (
          <ul className={rows.pastList}>
            {retired.map((one) => (
              <li className={rows.pastRow} key={one.id}>
                <span>{one.name}</span>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setEditing(one)}
                >
                  Edit
                </button>
              </li>
            ))}
          </ul>
        ) : null
      }
    >
      {current.length === 0 ? (
        <p className={rows.empty}>
          {kind === "position"
            ? "No position is defined. Illarin's jobs go here."
            : "Nothing is defined yet."}
        </p>
      ) : (
        <ol className={rows.list}>
          {current.map((one) => (
            <li
              className={rows.row}
              key={one.id}
              data-plain={kind === "position" || undefined}
            >
              {kind === "recognition" ? (
                <span
                  className={rows.mark}
                  data-blank={one.form === "title" || undefined}
                >
                  {one.mark ? (
                    <Image
                      src={one.mark.url}
                      alt=""
                      width={34}
                      height={34}
                      unoptimized
                    />
                  ) : null}
                </span>
              ) : null}
              <span className={rows.name}>{one.name}</span>
              <span className={rows.detail}>
                {one.explanation}
                {kind === "recognition" && !one.explanation ? (
                  <em className={styles.unexplained}>
                    Nobody is told what earns this
                  </em>
                ) : null}
              </span>
              <span className={rows.actions}>
                <button
                  type="button"
                  className={rows.iconButton}
                  onClick={() => reorder(one, -1)}
                  disabled={!movable(one, -1)}
                  aria-label={`Move ${one.name} up`}
                >
                  <ArrowUp size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={rows.iconButton}
                  onClick={() => reorder(one, 1)}
                  disabled={!movable(one, 1)}
                  aria-label={`Move ${one.name} down`}
                >
                  <ArrowDown size={15} strokeWidth={1.8} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={rows.textButton}
                  onClick={() => setEditing(one)}
                >
                  Edit
                </button>
              </span>
            </li>
          ))}
        </ol>
      )}

      {adding ? (
        <Dialog
          key="adding"
          open
          existing={null}
          onClose={() => setAdding(false)}
          onSaved={replace}
          onFailure={onFailure}
        />
      ) : null}
      {editing ? (
        <Dialog
          key={editing.id}
          open
          existing={editing}
          onClose={() => setEditing(null)}
          onSaved={replace}
          onFailure={onFailure}
        />
      ) : null}
    </Section>
  );
}
