"use client";

import { Field } from "@/components/ui/field";
import { Input, Textarea } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import type { LorebookEntry } from "@/lib/api/query";
import { CollectionStep } from "./workspace/CollectionStep";
import {
  moveItem,
  readLines,
  replaceAt,
  without,
  writeLines,
} from "./workspace/collection";
import { ChoiceField, FieldGroup, FieldPair } from "./workspace/fields";

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
      emptyMessage="No entries yet"
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
    <div className="flex flex-col gap-6">
      <Field hint="optional, and never sent to a model" label="Name">
        <Input
          disabled={pending}
          onChange={(event) =>
            onChange({ name: event.target.value || undefined })
          }
          value={entry.name ?? ""}
        />
      </Field>

      <Field label="Entry text">
        <Textarea
          disabled={pending}
          onChange={(event) => onChange({ text: event.target.value })}
          rows={10}
          value={entry.text}
        />
      </Field>

      <FieldGroup legend="Activation">
        <Field hint="one per line" label="Keys">
          <Textarea
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
          label="Enabled"
          onCheckedChange={(enabled) => onChange({ enabled })}
          disabled={pending}
        />
        <Switch
          checked={entry.constant ?? false}
          hint="On whatever the conversation says, keys or no keys."
          label="Always on"
          onCheckedChange={(constant) => onChange({ constant })}
          disabled={pending}
        />
        <Switch
          checked={entry.selective ?? false}
          hint="One of the keys below has to turn up too."
          label="Require a secondary key"
          onCheckedChange={(selective) => onChange({ selective })}
          disabled={pending}
        />
        <Field hint="one per line" label="Second keys">
          <Textarea
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
          hint='Off: "Dragon" matches "dragon".'
          label="Match the case of a key"
          onCheckedChange={(caseSensitive) => onChange({ caseSensitive })}
          disabled={pending}
        />
      </FieldGroup>

      <FieldGroup legend="Placement">
        <FieldPair>
          <Field hint="among the entries that fired with it" label="Order">
            <Input
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
              <option value="">Use app default</option>
              <option value="before_character">Before the character</option>
              <option value="after_character">After the character</option>
            </ChoiceField>
          </Field>
        </FieldPair>
      </FieldGroup>

      <FieldGroup legend="Recursive activation">
        <Switch
          checked={recursion.exclude ?? false}
          label="Don't let this entry switch others on"
          onCheckedChange={(exclude) =>
            onChange({ recursion: { ...recursion, exclude } })
          }
          disabled={pending}
        />
        <Switch
          checked={recursion.prevent ?? false}
          label="Don't let other entries switch this one on"
          onCheckedChange={(prevent) =>
            onChange({ recursion: { ...recursion, prevent } })
          }
          disabled={pending}
        />
        <Switch
          checked={recursion.delayUntil ?? false}
          label="Hold it back until a later pass"
          onCheckedChange={(delayUntil) =>
            onChange({ recursion: { ...recursion, delayUntil } })
          }
          disabled={pending}
        />
      </FieldGroup>
    </div>
  );
}
