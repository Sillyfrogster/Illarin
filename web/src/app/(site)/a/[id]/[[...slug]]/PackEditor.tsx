"use client";

import { ImagePlus, UserRound, X } from "lucide-react";
import Image from "next/image";
import { useMemo, useRef, useState } from "react";
import {
  addWorkImage,
  type LumiaRecord,
  type RecordListContent,
  type WorkImage,
} from "@/lib/api/query";
import { useWorkingCopy } from "@/lib/working-copy";
import { CollectionStep } from "./workspace/CollectionStep";
import { moveItem, replaceAt, without } from "./workspace/collection";
import {
  ChoiceField,
  Field,
  FieldGroup,
  FieldPair,
  TextAreaField,
  TextField,
} from "./workspace/fields";

const PRONOUNS: Array<{
  value: LumiaRecord["genderIdentity"];
  label: string;
}> = [
  { value: 0, label: "She / her" },
  { value: 1, label: "He / him" },
  { value: 2, label: "They / them" },
];

function recordName(record: LumiaRecord, position: number): string {
  return record.lumiaName.trim() || `Lumia ${position + 1}`;
}

export function PackEditor({
  workId,
  chosen,
  content,
  images,
  onChange,
  onChoose,
  onImageAdded,
  pending,
}: {
  workId: string;
  chosen: string | null;
  content: RecordListContent;
  images: WorkImage[];
  onChange: (content: RecordListContent) => void;
  onChoose: (key: string | null) => void;
  onImageAdded: () => void;
  pending: boolean;
}) {
  const records = content.records;

  return (
    <CollectionStep
      chosen={chosen}
      emptyMessage="This pack has no Lumia yet."
      noun="Lumia"
      onAdd={() =>
        onChange({
          ...content,
          records: [
            ...records,
            {
              authorName: "",
              genderIdentity: 2,
              lumiaBehavior: "",
              lumiaDefinition: "",
              lumiaName: "",
              lumiaPersonality: "",
              version: 1,
            },
          ],
        })
      }
      onChoose={onChoose}
      onMove={(from, to) =>
        onChange({ ...content, records: moveItem(records, from, to) })
      }
      onRemove={(index) =>
        onChange({ ...content, records: without(records, index) })
      }
      pending={pending}
      plural="Lumia"
      rows={records.map((record, index) => ({
        detail: record.authorName.trim() || "No author named",
        id: record.id,
        name: recordName(record, index),
        search: [record.lumiaName, record.authorName, record.lumiaDefinition]
          .join(" ")
          .toLowerCase(),
      }))}
    >
      {(index) => (
        <LumiaFields
          workId={workId}
          images={images}
          onChange={(changes) =>
            onChange({
              ...content,
              records: replaceAt(records, index, changes),
            })
          }
          onImageAdded={onImageAdded}
          pending={pending}
          record={records[index]}
        />
      )}
    </CollectionStep>
  );
}

function LumiaFields({
  workId,
  images,
  onChange,
  onImageAdded,
  pending,
  record,
}: {
  workId: string;
  images: WorkImage[];
  onChange: (changes: Partial<LumiaRecord>) => void;
  onImageAdded: () => void;
  pending: boolean;
  record: LumiaRecord;
}) {
  return (
    <div className="flex flex-col gap-6">
      <AvatarField
        workId={workId}
        images={images}
        onChange={onChange}
        onImageAdded={onImageAdded}
        pending={pending}
        record={record}
      />

      <Field label="Name">
        <TextField
          disabled={pending}
          onChange={(event) => onChange({ lumiaName: event.target.value })}
          value={record.lumiaName}
        />
      </Field>

      <FieldGroup legend="Credit and identity">
        <FieldPair>
          <Field label="Author">
            <TextField
              disabled={pending}
              onChange={(event) => onChange({ authorName: event.target.value })}
              value={record.authorName}
            />
          </Field>
          <Field label="Version">
            <TextField
              disabled={pending}
              min={1}
              onChange={(event) =>
                onChange({ version: Math.max(1, Number(event.target.value)) })
              }
              step={1}
              type="number"
              value={record.version}
            />
          </Field>
        </FieldPair>
        <Field label="Pronouns">
          <ChoiceField
            disabled={pending}
            onChange={(event) =>
              onChange({
                genderIdentity: Number(
                  event.target.value,
                ) as LumiaRecord["genderIdentity"],
              })
            }
            value={record.genderIdentity}
          >
            {PRONOUNS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </ChoiceField>
        </Field>
      </FieldGroup>

      <Field label="Definition">
        <TextAreaField
          disabled={pending}
          onChange={(event) =>
            onChange({ lumiaDefinition: event.target.value })
          }
          rows={7}
          value={record.lumiaDefinition}
        />
      </Field>
      <Field label="Personality">
        <TextAreaField
          disabled={pending}
          onChange={(event) =>
            onChange({ lumiaPersonality: event.target.value })
          }
          rows={7}
          value={record.lumiaPersonality}
        />
      </Field>
      <Field label="Behaviour">
        <TextAreaField
          disabled={pending}
          onChange={(event) => onChange({ lumiaBehavior: event.target.value })}
          rows={7}
          value={record.lumiaBehavior}
        />
      </Field>
    </div>
  );
}

function AvatarField({
  workId,
  images,
  onChange,
  onImageAdded,
  pending,
  record,
}: {
  workId: string;
  images: WorkImage[];
  onChange: (changes: Partial<LumiaRecord>) => void;
  onImageAdded: () => void;
  pending: boolean;
  record: LumiaRecord;
}) {
  const candidate = useWorkingCopy();
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState("");
  const [preview, setPreview] = useState("");
  const file = useRef<HTMLInputElement>(null);
  const imagesById = useMemo(
    () => new Map(images.map((image) => [image.id, image])),
    [images],
  );
  const stored = record.avatarUrl
    ? imagesById.get(record.avatarUrl)
    : undefined;
  const source = preview || stored?.thumbUrl;
  const missing = Boolean(record.avatarUrl) && !source;

  async function upload(chosen: File | null) {
    if (!chosen || uploading) return;
    setUploading(true);
    setMessage("");
    try {
      const mediaId = await addWorkImage(
        candidate,
        workId,
        chosen,
        "pack_item",
      );
      setPreview(URL.createObjectURL(chosen));
      onChange({ avatarUrl: mediaId });
      onImageAdded();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "The avatar could not be added. Try again.",
      );
    } finally {
      setUploading(false);
      if (file.current) file.current.value = "";
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-4">
      <div className="grid size-24 shrink-0 place-items-center overflow-hidden rounded-plate bg-media text-on-media">
        {source ? (
          <Image
            alt=""
            height={stored?.height ?? 240}
            sizes="120px"
            src={source}
            unoptimized
            width={stored?.width ?? 240}
          />
        ) : (
          <UserRound aria-hidden="true" size={34} strokeWidth={1.35} />
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <p className="text-label font-medium text-mute">Avatar</p>
        <p className="text-meta text-mute">
          {missing
            ? "This avatar is missing. Upload an image to replace it."
            : "Square images work best. Uploading an avatar does not change the original file."}
        </p>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
        <div className="flex flex-wrap items-center gap-1">
          <label className="inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-control bg-deep px-4 text-meta font-medium text-ink hover:bg-rule/45 has-disabled:opacity-45">
            <ImagePlus aria-hidden="true" size={16} />
            {uploading ? "Adding…" : source ? "Replace avatar" : "Add avatar"}
            <input
              accept="image/*"
              className="sr-only"
              disabled={pending || uploading}
              onChange={(event) => void upload(event.target.files?.[0] ?? null)}
              ref={file}
              type="file"
            />
          </label>
          {record.avatarUrl ? (
            <button
              className="inline-flex min-h-11 items-center gap-1.5 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink disabled:opacity-45"
              disabled={pending || uploading}
              onClick={() => {
                setPreview("");
                onChange({ avatarUrl: undefined });
              }}
              type="button"
            >
              <X aria-hidden="true" size={15} />
              Remove avatar
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
