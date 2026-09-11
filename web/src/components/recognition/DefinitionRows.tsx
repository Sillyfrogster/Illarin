"use client";

import { ArrowDown, ArrowUp, ImageUp, Plus, Trash2, Type } from "lucide-react";
import Image from "next/image";
import { useRef, useState } from "react";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  Row,
  RowMark,
  RowMove,
  Rows,
  StartAction,
} from "@/components/register/RowParts";
import { Consequence, StepForm } from "@/components/register/StepParts";
import { Field, TextInput } from "@/components/ui/field";
import {
  clearMark,
  defineDistinction,
  orderDistinctions,
  updateDistinction,
  uploadMark,
} from "@/lib/api/distinctions";
import type { Distinction } from "@/lib/api/query";
import {
  movedDefinitions,
  nothingIn,
  type RecognitionRegister,
  registerHolds,
  registerName,
  unexplained,
} from "@/lib/recognition-register";

const NAME_LIMIT = 48;
const EARNED_LIMIT = 200;

export function DefinitionRows({
  definitions,
  onChanged,
  onFailure,
  onOpen,
  register,
}: {
  definitions: Distinction[];
  onChanged: (definitions: Distinction[]) => void;
  onFailure: (message: string) => void;
  onOpen: (existing: Distinction | null) => void;
  register: Exclude<RecognitionRegister, "accounts">;
}) {
  const { current, retired } = registerHolds(register, definitions);
  const marked = register === "titles";

  async function reorder(one: Distinction, step: number) {
    const order = movedDefinitions(definitions, one, step);
    if (!order) return;
    const answer = await orderDistinctions(one.form, order);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(answer.value.definitions);
  }

  async function bringBack(one: Distinction) {
    const answer = await updateDistinction(one.id, { retired: false });
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    onChanged(
      definitions.map((other) =>
        other.id === answer.value?.id ? answer.value : other,
      ),
    );
  }

  return (
    <>
      <PanelHead
        action={
          <StartAction icon={Plus} onClick={() => onOpen(null)}>
            {marked ? "Add a title or badge" : "Add a position"}
          </StartAction>
        }
        id="register-heading"
        title={registerName(register)}
      />

      {current.length === 0 ? (
        <Nothing>{nothingIn(register)}</Nothing>
      ) : (
        <Rows>
          {current.map((one) => (
            <Row
              aside={
                <>
                  <RowMove
                    disabled={!movedDefinitions(definitions, one, -1)}
                    icon={ArrowUp}
                    label={`Move ${one.name} up`}
                    onClick={() => reorder(one, -1)}
                  />
                  <RowMove
                    disabled={!movedDefinitions(definitions, one, 1)}
                    icon={ArrowDown}
                    label={`Move ${one.name} down`}
                    onClick={() => reorder(one, 1)}
                  />
                </>
              }
              key={one.id}
              lead={marked ? <DistinctionMark one={one} /> : undefined}
              onOpen={() => onOpen(one)}
              open={`Edit ${one.name}`}
              standing={
                unexplained(one) ? (
                  <span className="text-stop">No award criteria provided</span>
                ) : (
                  one.explanation || undefined
                )
              }
              title={one.name}
            />
          ))}
        </Rows>
      )}

      {retired.length > 0 ? (
        <Past summary={`${retired.length} retired`}>
          {retired.map((one) => (
            <PastRow
              action={
                <button
                  className="inline-flex min-h-11 items-center rounded-control px-3 font-ui text-meta font-medium text-accent outline-offset-3 hover:underline"
                  onClick={() => bringBack(one)}
                  type="button"
                >
                  Reactivate
                </button>
              }
              key={one.id}
            >
              {one.name}
            </PastRow>
          ))}
        </Past>
      ) : null}
    </>
  );
}

function DistinctionMark({ one }: { one: Distinction }) {
  return (
    <RowMark tone={one.mark ? "accent" : "quiet"}>
      {one.mark ? (
        <Image
          alt=""
          height={44}
          src={one.mark.url}
          unoptimized
          width={44}
          className="size-11 object-contain"
        />
      ) : (
        <Type aria-hidden="true" className="size-4.5" strokeWidth={1.7} />
      )}
    </RowMark>
  );
}

export function DefinitionStep({
  existing,
  onClose,
  onFailure,
  onSaved,
  register,
}: {
  existing: Distinction | null;
  onClose: () => void;
  onFailure: (message: string) => void;
  onSaved: (saved: Distinction, added: boolean) => void;
  register: Exclude<RecognitionRegister, "accounts">;
}) {
  const marked = register === "titles";
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
      ? await updateDistinction(
          existing.id,
          marked
            ? { explanation: earned.trim(), name: name.trim() }
            : { name: name.trim() },
        )
      : await defineDistinction(
          marked ? "title" : "position",
          name.trim(),
          marked ? earned.trim() : "",
        );
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    let saved = written.value;
    if (chosen) {
      const withMark = await uploadMark(saved.id, chosen);
      if (withMark.error || !withMark.value) {
        setBusy(false);
        onSaved(saved, !existing);
        onFailure(withMark.error ?? "");
        return;
      }
      saved = withMark.value;
    }
    setBusy(false);
    onSaved(saved, !existing);
    onClose();
  }

  async function retire() {
    if (!existing) return;
    setBusy(true);
    const answer = await updateDistinction(existing.id, { retired: true });
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
    <StepForm
      busy={busy}
      commit={existing ? "Save" : "Add"}
      onCommit={save}
      ready={Boolean(name.trim())}
      under={
        existing ? (
          <Consequence
            action="Retire"
            busy={busy}
            confirm="Retire recognition"
            onConfirm={retire}
          >
            Nobody new gets {existing.name}. Everyone holding it keeps it, and
            their profile still shows it.
          </Consequence>
        ) : null
      }
    >
      <Field
        hint={
          marked
            ? undefined
            : "A job somebody does at Illarin. Every position an account holds shows on their profile."
        }
        htmlFor="definition-name"
        label="Name"
      >
        <TextInput
          id="definition-name"
          maxLength={NAME_LIMIT}
          onChange={(event) => setName(event.target.value)}
          placeholder={marked ? "First light" : "Developer"}
          value={name}
        />
      </Field>

      {marked ? (
        <>
          <Field
            hint="Written for the person reading a profile. Say what someone did to get it."
            htmlFor="definition-earned"
            label="Award criteria"
          >
            <TextInput
              id="definition-earned"
              maxLength={EARNED_LIMIT}
              onChange={(event) => setEarned(event.target.value)}
              placeholder="Published a first asset"
              value={earned}
            />
          </Field>

          <Field
            hint="With a mark it shows as a badge. Without one it shows as a title."
            label="Mark"
          >
            <div className="flex flex-wrap items-center gap-3">
              <span className="grid size-14 shrink-0 place-items-center overflow-hidden rounded-control bg-deep font-ui text-label text-mute">
                {shown ? (
                  <Image
                    alt=""
                    className="size-14 object-contain"
                    height={56}
                    src={shown}
                    unoptimized
                    width={56}
                  />
                ) : (
                  "No mark"
                )}
              </span>
              <button
                className="inline-flex min-h-11 items-center gap-2 rounded-control bg-deep px-4 font-ui text-ui font-medium text-ink outline-offset-3 hover:bg-rule/45"
                onClick={() => fileInput.current?.click()}
                type="button"
              >
                <ImageUp
                  aria-hidden="true"
                  className="size-4"
                  strokeWidth={1.9}
                />
                {shown ? "Replace" : "Upload image"}
              </button>
              {shown ? (
                <button
                  className="inline-flex min-h-11 items-center gap-2 rounded-control bg-stop-wash px-4 font-ui text-ui font-medium text-stop outline-offset-3 hover:opacity-85"
                  onClick={drop}
                  type="button"
                >
                  <Trash2
                    aria-hidden="true"
                    className="size-4"
                    strokeWidth={1.9}
                  />
                  Remove
                </button>
              ) : null}
              <input
                accept="image/png,image/jpeg,image/webp,image/gif"
                className="sr-only"
                onChange={(event) => take(event.target.files?.[0])}
                ref={fileInput}
                type="file"
              />
            </div>
          </Field>
        </>
      ) : null}
    </StepForm>
  );
}
