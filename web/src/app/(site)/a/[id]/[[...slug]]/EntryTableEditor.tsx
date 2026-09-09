"use client";

import type { LorebookEntry } from "@/lib/api/query";
import { CollectionStep } from "./workspace/CollectionStep";
import {
  moveItem,
  readLines,
  replaceAt,
  without,
  writeLines,
} from "./workspace/collection";
import {
  ChoiceField,
  Field,
  FieldGroup,
  FieldPair,
  Switch,
  TextAreaField,
  TextField,
} from "./workspace/fields";

export function entryName(entry: LorebookEntry, position: number): string {
  if (entry.name?.trim()) return entry.name;
  if (entry.keys.length > 0) return entry.keys.join(", ");
  return `Entry ${position + 1}`;
}

export function EntryTableEditor({
  chosen,
  entries,
  onChange,
  onChoose,
  pending,
}: {
  chosen: string | null;
  entries: LorebookEntry[];
  onChange: (entries: LorebookEntry[]) => void;
  onChoose: (key: string | null) => void;
  pending: boolean;
}) {
  return (
    <CollectionStep
      chosen={chosen}
      emptyMessage="This book has no entries yet."
      noun="entry"
      onAdd={() =>
        onChange([...entries, { enabled: true, keys: [], text: "" }])
      }
      onChoose={onChoose}
      onMove={(from, to) => onChange(moveItem(entries, from, to))}
      onRemove={(index) => onChange(without(entries, index))}
      pending={pending}
      plural="entries"
      rows={entries.map((entry, index) => ({
        detail:
          entry.keys.length === 0
            ? "No keys"
            : entry.keys.slice(0, 4).join(" · "),
        id: entry.id,
        name: entryName(entry, index),
        off: !entry.enabled,
        search: [entryName(entry, index), entry.text, entry.keys.join(" ")]
          .join(" ")
          .toLowerCase(),
      }))}
    >
      {(index) => (
        <EntryFields
          entry={entries[index]}
          onChange={(changes) => onChange(replaceAt(entries, index, changes))}
          pending={pending}
        />
      )}
    </CollectionStep>
  );
}

function EntryFields({
  entry,
  onChange,
  pending,
}: {
  entry: LorebookEntry;
  onChange: (changes: Partial<LorebookEntry>) => void;
  pending: boolean;
}) {
  const recursion = entry.recursion ?? {};
  return (
    <div className="space-y-6">
      <Field hint="optional, and never sent to a model" label="Name">
        <TextField
          disabled={pending}
          onChange={(event) =>
            onChange({ name: event.target.value || undefined })
          }
          value={entry.name ?? ""}
        />
      </Field>

      <Field label="Entry text">
        <TextAreaField
          disabled={pending}
          onChange={(event) => onChange({ text: event.target.value })}
          rows={10}
          value={entry.text}
        />
      </Field>

      <FieldGroup legend="What switches it on">
        <Field hint="one per line" label="Keys">
          <TextAreaField
            disabled={pending}
            onChange={(event) =>
              onChange({ keys: readLines(event.target.value) })
            }
            rows={4}
            value={writeLines(entry.keys)}
          />
        </Field>
        <Switch
          checked={entry.enabled}
          hint="A switched-off entry stays in the book and reaches no model."
          label="Switched on"
          onChange={(enabled) => onChange({ enabled })}
          pending={pending}
        />
        <Switch
          checked={entry.constant ?? false}
          hint="On whatever the conversation says, keys or no keys."
          label="Always on"
          onChange={(constant) => onChange({ constant })}
          pending={pending}
        />
        <Switch
          checked={entry.selective ?? false}
          hint="One of the keys below has to turn up too."
          label="Needs a second key as well"
          onChange={(selective) => onChange({ selective })}
          pending={pending}
        />
        <Field hint="one per line" label="Second keys">
          <TextAreaField
            disabled={pending}
            onChange={(event) =>
              onChange({ secondaryKeys: readLines(event.target.value) })
            }
            rows={3}
            value={writeLines(entry.secondaryKeys)}
          />
        </Field>
        <Switch
          checked={entry.caseSensitive ?? false}
          hint="Off, ledger and Ledger both count."
          label="Match the case of a key"
          onChange={(caseSensitive) => onChange({ caseSensitive })}
          pending={pending}
        />
      </FieldGroup>

      <FieldGroup legend="Where it goes">
        <FieldPair>
          <Field hint="among the entries that fired with it" label="Order">
            <TextField
              disabled={pending}
              onChange={(event) =>
                onChange({ order: Number(event.target.value) || 0 })
              }
              type="number"
              value={entry.order ?? 0}
            />
          </Field>
          <Field label="Position">
            <ChoiceField
              disabled={pending}
              onChange={(event) =>
                onChange({
                  position:
                    event.target.value === ""
                      ? undefined
                      : (event.target.value as LorebookEntry["position"]),
                })
              }
              value={entry.position ?? ""}
            >
              <option value="">Leave it to whatever reads the book</option>
              <option value="before_character">Before the character</option>
              <option value="after_character">After the character</option>
            </ChoiceField>
          </Field>
        </FieldPair>
      </FieldGroup>

      <FieldGroup legend="Passes after the first">
        <Switch
          checked={recursion.exclude ?? false}
          label="Do not let this entry switch others on"
          onChange={(exclude) =>
            onChange({ recursion: { ...recursion, exclude } })
          }
          pending={pending}
        />
        <Switch
          checked={recursion.prevent ?? false}
          label="Do not let other entries switch this one on"
          onChange={(prevent) =>
            onChange({ recursion: { ...recursion, prevent } })
          }
          pending={pending}
        />
        <Switch
          checked={recursion.delayUntil ?? false}
          label="Hold it back until a later pass"
          onChange={(delayUntil) =>
            onChange({ recursion: { ...recursion, delayUntil } })
          }
          pending={pending}
        />
      </FieldGroup>
    </div>
  );
}
