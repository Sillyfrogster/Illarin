"use client";

import { X } from "lucide-react";
import { useState } from "react";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import type {
  PresetSetting,
  PresetVariable,
  PromptFragment,
  PromptGroup,
  PromptListContent,
  RegexScript,
  TypedValue,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { nameSlot } from "@/lib/preset-slots";
import { CollectionStep } from "./workspace/CollectionStep";
import {
  moveItem,
  readLines,
  replaceAt,
  without,
  writeLines,
} from "./workspace/collection";
import {
  AddAction,
  ChoiceField,
  Field,
  FieldGroup,
  FieldPair,
  Note,
  RemoveAction,
  Switch,
  TextAreaField,
  TextField,
} from "./workspace/fields";

const PROMPT_ROLES: {
  value: NonNullable<PromptFragment["role"]>;
  label: string;
}[] = [
  { value: "system", label: "System" },
  { value: "user", label: "User" },
  { value: "assistant", label: "Assistant" },
  { value: "user_append", label: "User, added to the message before it" },
  {
    value: "assistant_append",
    label: "Assistant, added to the message before it",
  },
];

const PLACEMENTS: {
  value: NonNullable<PromptFragment["placement"]>;
  label: string;
}[] = [
  { value: "pre_history", label: "Before the conversation" },
  { value: "in_history", label: "Inside the conversation" },
  { value: "post_history", label: "After the conversation" },
];

const WIDGETS: { value: PresetVariable["widget"]; label: string }[] = [
  { value: "switch", label: "Yes or no" },
  { value: "select", label: "Single choice" },
  { value: "multiselect", label: "Multiple choice" },
  { value: "number", label: "A number" },
  { value: "slider", label: "A number on a slider" },
  { value: "text", label: "A line of text" },
  { value: "textarea", label: "Several lines of text" },
];

const SCRIPT_TARGETS: {
  value: NonNullable<RegexScript["targets"]>[number];
  label: string;
}[] = [
  { value: "user_input", label: "What the reader writes" },
  { value: "model_output", label: "What the model writes back" },
  { value: "slash_command", label: "What a command produces" },
  { value: "lorebook", label: "What the lorebook adds" },
];

const SCRIPT_EFFECTS: {
  value: NonNullable<RegexScript["affects"]>[number];
  label: string;
}[] = [
  { value: "display", label: "What a person is shown" },
  { value: "prompt", label: "What the model is sent" },
];

export function fragmentName(
  fragment: PromptFragment,
  position: number,
): string {
  if (fragment.name?.trim()) return fragment.name;
  if (fragment.marker?.trim()) return fragment.marker;
  return `Fragment ${position + 1}`;
}

export function PromptListEditor({
  chosen,
  content,
  onChange,
  onChoose,
  pending,
}: {
  chosen: string | null;
  content: PromptListContent;
  onChange: (content: PromptListContent) => void;
  onChoose: (key: string | null) => void;
  pending: boolean;
}) {
  const groups = content.groups ?? [];
  const fragments = content.fragments ?? [];
  const groupNames = new Map(groups.map((group) => [group.id, group.name]));

  return (
    <CollectionStep
      above={
        <GroupEditor
          fragments={fragments}
          groups={groups}
          onChange={onChange}
          pending={pending}
        />
      }
      chosen={chosen}
      emptyMessage="This preset has no prompt fragments yet."
      noun="fragment"
      onAdd={() =>
        onChange({
          fragments: [
            ...fragments,
            { enabled: true, role: "system", text: "" },
          ],
          groups,
        })
      }
      onChoose={onChoose}
      onMove={(from, to) =>
        onChange({ fragments: moveItem(fragments, from, to), groups })
      }
      onRemove={(index) =>
        onChange({ fragments: without(fragments, index), groups })
      }
      pending={pending}
      rows={fragments.map((fragment, index) => ({
        detail: fragment.groupId ? groupNames.get(fragment.groupId) : undefined,
        id: fragment.id,
        name: fragmentName(fragment, index),
        off: !fragment.enabled,
        private: fragment.private ?? false,
        search: [
          fragmentName(fragment, index),
          fragment.text,
          fragment.marker ?? "",
        ]
          .join(" ")
          .toLowerCase(),
      }))}
    >
      {(index) => (
        <FragmentFields
          fragment={fragments[index]}
          groups={groups}
          onChange={(changes) =>
            onChange({
              fragments: replaceAt(fragments, index, changes),
              groups,
            })
          }
          pending={pending}
        />
      )}
    </CollectionStep>
  );
}

function GroupEditor({
  fragments,
  groups,
  onChange,
  pending,
}: {
  fragments: PromptFragment[];
  groups: PromptGroup[];
  onChange: (content: PromptListContent) => void;
  pending: boolean;
}) {
  const [adding, setAdding] = useState("");

  function addGroup() {
    if (adding.trim() === "") return;
    onChange({
      fragments,
      groups: [...groups, { id: crypto.randomUUID(), name: adding.trim() }],
    });
    setAdding("");
  }

  function removeGroup(index: number) {
    const gone = groups[index].id;
    onChange({
      fragments: fragments.map((fragment) =>
        fragment.groupId === gone
          ? { ...fragment, groupId: undefined }
          : fragment,
      ),
      groups: without(groups, index),
    });
  }

  return (
    <MorphingDisclosure
      summary={
        groups.length === 0
          ? "No headings"
          : `${groups.length} ${groups.length === 1 ? "heading" : "headings"}`
      }
    >
      <div className="flex flex-col gap-3 pt-3">
        {groups.length === 0 ? (
          <Note>
            Fragments run in one list until you add a heading to group them
            under.
          </Note>
        ) : (
          <ul className="flex flex-col gap-2">
            {groups.map((group, index) => (
              <li className="flex items-center gap-1" key={group.id ?? index}>
                <TextField
                  aria-label={`Heading ${index + 1}`}
                  disabled={pending}
                  onChange={(event) =>
                    onChange({
                      fragments,
                      groups: replaceAt(groups, index, {
                        name: event.target.value,
                      }),
                    })
                  }
                  value={group.name}
                />
                <button
                  aria-label={`Remove the heading ${group.name}`}
                  className="grid size-11 shrink-0 place-items-center rounded-control text-mute outline-offset-3 hover:bg-stop-wash hover:text-stop disabled:opacity-45"
                  disabled={pending}
                  onClick={() => removeGroup(index)}
                  type="button"
                >
                  <X aria-hidden="true" size={16} />
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="flex flex-wrap items-center gap-2">
          <TextField
            aria-label="A new heading"
            className="min-w-40 flex-1"
            disabled={pending}
            onChange={(event) => setAdding(event.target.value)}
            onKeyDown={(event) => {
              if (event.key !== "Enter") return;
              event.preventDefault();
              addGroup();
            }}
            placeholder="A new heading"
            value={adding}
          />
          <AddAction
            disabled={pending || adding.trim() === ""}
            onClick={addGroup}
          >
            Add heading
          </AddAction>
        </div>
      </div>
    </MorphingDisclosure>
  );
}

function FragmentFields({
  fragment,
  groups,
  onChange,
  pending,
}: {
  fragment: PromptFragment;
  groups: PromptGroup[];
  onChange: (changes: Partial<PromptFragment>) => void;
  pending: boolean;
}) {
  const isMarker = (fragment.marker ?? "") !== "";
  return (
    <div className="flex flex-col gap-6">
      <Field hint="optional, and never sent to a model" label="Name">
        <TextField
          disabled={pending}
          onChange={(event) =>
            onChange({ name: event.target.value || undefined })
          }
          value={fragment.name ?? ""}
        />
      </Field>

      {!isMarker ? (
        <Switch
          checked={fragment.private ?? false}
          hint="Its text is sent only to an allowed connected app."
          label="Private prompt"
          onChange={(isPrivate) => onChange({ private: isPrivate })}
          pending={pending}
        />
      ) : null}

      {isMarker ? (
        <Note>
          This fragment is a marker. The app splices its own content in here,
          named <code className="font-mono">{fragment.marker}</code>, so it
          carries no text.
        </Note>
      ) : (
        <Field label="Fragment text">
          <TextAreaField
            disabled={pending}
            onChange={(event) => onChange({ text: event.target.value })}
            rows={12}
            value={fragment.text}
          />
        </Field>
      )}

      <FieldGroup legend="How it is sent">
        <Field label="Message role">
          <ChoiceField
            disabled={pending}
            onChange={(event) =>
              onChange({
                role:
                  event.target.value === ""
                    ? undefined
                    : (event.target.value as PromptFragment["role"]),
              })
            }
            value={fragment.role ?? ""}
          >
            <option value="">Use app default</option>
            {PROMPT_ROLES.map((role) => (
              <option key={role.value} value={role.value}>
                {role.label}
              </option>
            ))}
          </ChoiceField>
        </Field>
        <Field label="Heading">
          <ChoiceField
            disabled={pending}
            onChange={(event) =>
              onChange({ groupId: event.target.value || undefined })
            }
            value={fragment.groupId ?? ""}
          >
            <option value="">No heading</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </ChoiceField>
        </Field>
        <Switch
          checked={fragment.enabled}
          hint="A switched-off fragment stays in the preset and reaches no model."
          label="Enabled"
          onChange={(enabled) => onChange({ enabled })}
          pending={pending}
        />
      </FieldGroup>

      <FieldGroup legend="Placement">
        <Field label="Placement">
          <ChoiceField
            disabled={pending}
            onChange={(event) =>
              onChange({
                placement:
                  event.target.value === ""
                    ? undefined
                    : (event.target.value as PromptFragment["placement"]),
              })
            }
            value={fragment.placement ?? ""}
          >
            <option value="">Use app default</option>
            {PLACEMENTS.map((placement) => (
              <option key={placement.value} value={placement.value}>
                {placement.label}
              </option>
            ))}
          </ChoiceField>
        </Field>
        <Field hint="messages back from the most recent" label="Depth">
          <TextField
            disabled={pending}
            onChange={(event) =>
              onChange({
                depth:
                  event.target.value === ""
                    ? undefined
                    : Number(event.target.value),
              })
            }
            type="number"
            value={fragment.depth ?? ""}
          />
        </Field>
      </FieldGroup>
    </div>
  );
}

export function SettingGroupEditor({
  onChange,
  pending,
  settings,
}: {
  onChange: (settings: PresetSetting[]) => void;
  pending: boolean;
  settings: PresetSetting[];
}) {
  return (
    <div className="flex flex-col gap-4">
      {settings.length === 0 ? (
        <Note>
          This group has no settings yet. Add the names your app reads.
        </Note>
      ) : null}
      {settings.map((setting, index) => (
        <SettingRow
          key={setting.id ?? setting.name}
          onChange={(changes) => onChange(replaceAt(settings, index, changes))}
          onRemove={() => onChange(without(settings, index))}
          pending={pending}
          setting={setting}
        />
      ))}
      <NewSetting
        onAdd={(setting) => onChange([...settings, setting])}
        pending={pending}
      />
    </div>
  );
}

function SettingRow({
  onChange,
  onRemove,
  pending,
  setting,
}: {
  onChange: (changes: Partial<PresetSetting>) => void;
  onRemove: () => void;
  pending: boolean;
  setting: PresetSetting;
}) {
  const supplied = setting.value != null;
  const slot = nameSlot(setting.name);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5">
        <span
          className={cn(
            "text-ui font-medium wrap-anywhere",
            supplied ? "text-ink" : "text-mute",
          )}
        >
          {slot.name}
        </span>
        <span className="flex flex-wrap gap-x-2 text-label text-mute">
          {slot.rank === "unrecognised" ? null : (
            <code className="font-mono wrap-anywhere">{setting.name}</code>
          )}
          {setting.type.replace("_", " ")}
        </span>
      </div>
      <ValueField
        choices={setting.choices}
        label={setting.name}
        onChange={(value) => onChange({ value })}
        pending={pending}
        type={setting.type}
        value={setting.value}
      />
      <div className="flex flex-wrap items-center gap-1">
        <button
          className="inline-flex min-h-11 items-center rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink disabled:opacity-45"
          disabled={pending}
          onClick={() =>
            onChange({
              value: supplied ? undefined : emptyValue(setting.type),
            })
          }
          type="button"
        >
          {supplied ? "Leave it out" : "Fill it in"}
        </button>
        <RemoveAction
          disabled={pending}
          label={`Remove ${setting.name}`}
          onClick={onRemove}
        />
      </div>
    </div>
  );
}

function NewSetting({
  onAdd,
  pending,
}: {
  onAdd: (setting: PresetSetting) => void;
  pending: boolean;
}) {
  const [name, setName] = useState("");
  const [type, setType] = useState<PresetSetting["type"]>("number");
  return (
    <div className="flex flex-wrap items-center gap-2 pt-2">
      <TextField
        aria-label="A new setting's name"
        className="max-w-64 flex-1"
        disabled={pending}
        onChange={(event) => setName(event.target.value)}
        placeholder="The name your app reads"
        value={name}
      />
      <ChoiceField
        aria-label="What the new setting holds"
        className="max-w-44 flex-1"
        disabled={pending}
        onChange={(event) =>
          setType(event.target.value as PresetSetting["type"])
        }
        value={type}
      >
        <option value="number">A number</option>
        <option value="boolean">Yes or no</option>
        <option value="text">Text</option>
        <option value="string_list">A list of strings</option>
      </ChoiceField>
      <AddAction
        disabled={pending || name.trim() === ""}
        onClick={() => {
          if (name.trim() === "") return;
          onAdd({ name: name.trim(), type });
          setName("");
        }}
      >
        Add setting
      </AddAction>
    </div>
  );
}

function emptyValue(type: PresetSetting["type"]): TypedValue {
  switch (type) {
    case "number":
      return { number: 0 };
    case "boolean":
      return { boolean: false };
    case "string_list":
      return { strings: [] };
    default:
      return { text: "" };
  }
}

function ValueField({
  choices,
  label,
  onChange,
  pending,
  type,
  value,
}: {
  choices?: string[];
  label: string;
  onChange: (value: TypedValue) => void;
  pending: boolean;
  type: PresetSetting["type"];
  value: TypedValue | undefined;
}) {
  if (value == null) {
    return (
      <p className="font-display text-ui text-mute italic">No value set.</p>
    );
  }
  if (type === "boolean") {
    return (
      <Switch
        checked={value.boolean ?? false}
        label={value.boolean ? "Yes" : "No"}
        onChange={(boolean) => onChange({ boolean })}
        pending={pending}
      />
    );
  }
  if (type === "number") {
    return (
      <TextField
        aria-label={label}
        disabled={pending}
        onChange={(event) =>
          onChange({
            number: event.target.value === "" ? 0 : Number(event.target.value),
          })
        }
        type="number"
        value={value.number ?? ""}
      />
    );
  }
  if (type === "string_list") {
    return (
      <TextAreaField
        aria-label={`${label}, one per line`}
        disabled={pending}
        onChange={(event) =>
          onChange({ strings: readLines(event.target.value) })
        }
        rows={3}
        value={writeLines(value.strings)}
      />
    );
  }
  if (choices && choices.length > 0) {
    return (
      <ChoiceField
        aria-label={label}
        disabled={pending}
        onChange={(event) => onChange({ text: event.target.value })}
        value={value.text ?? ""}
      >
        {choices.map((choice) => (
          <option key={choice} value={choice}>
            {choice}
          </option>
        ))}
      </ChoiceField>
    );
  }
  return (
    <TextField
      aria-label={label}
      disabled={pending}
      onChange={(event) => onChange({ text: event.target.value })}
      value={value.text ?? ""}
    />
  );
}

export function variableName(
  variable: PresetVariable,
  position: number,
): string {
  if (variable.label?.trim()) return variable.label;
  if (variable.name.trim()) return variable.name;
  return `Variable ${position + 1}`;
}

export function VariableSchemaEditor({
  chosen,
  onChange,
  onChoose,
  pending,
  variables,
}: {
  chosen: string | null;
  onChange: (variables: PresetVariable[]) => void;
  onChoose: (key: string | null) => void;
  pending: boolean;
  variables: PresetVariable[];
}) {
  return (
    <CollectionStep
      chosen={chosen}
      emptyMessage="No variables yet. Add a variable to let readers customise prompt values."
      noun="variable"
      onAdd={() => onChange([...variables, { name: "", widget: "switch" }])}
      onChoose={onChoose}
      onMove={(from, to) => onChange(moveItem(variables, from, to))}
      onRemove={(index) => onChange(without(variables, index))}
      pending={pending}
      rows={variables.map((variable, index) => ({
        detail: variable.name,
        id: variable.id,
        name: variableName(variable, index),
        search: [
          variable.name,
          variable.label ?? "",
          variable.description ?? "",
        ]
          .join(" ")
          .toLowerCase(),
      }))}
    >
      {(index) => (
        <VariableFields
          onChange={(changes) => onChange(replaceAt(variables, index, changes))}
          pending={pending}
          variable={variables[index]}
        />
      )}
    </CollectionStep>
  );
}

function VariableFields({
  onChange,
  pending,
  variable,
}: {
  onChange: (changes: Partial<PresetVariable>) => void;
  pending: boolean;
  variable: PresetVariable;
}) {
  const options = variable.options ?? [];
  const range = variable.range ?? {};
  const listed =
    variable.widget === "select" || variable.widget === "multiselect";
  const numeric = variable.widget === "number" || variable.widget === "slider";
  return (
    <div className="flex flex-col gap-6">
      <Field hint="what the fragments refer to it by" label="Name">
        <TextField
          disabled={pending}
          onChange={(event) => onChange({ name: event.target.value })}
          value={variable.name}
        />
      </Field>

      <Field label="Input type">
        <ChoiceField
          disabled={pending}
          onChange={(event) =>
            onChange({ widget: event.target.value as PresetVariable["widget"] })
          }
          value={variable.widget}
        >
          {WIDGETS.map((widget) => (
            <option key={widget.value} value={widget.value}>
              {widget.label}
            </option>
          ))}
        </ChoiceField>
      </Field>

      <Field hint="what a reader sees above the control" label="Label">
        <TextField
          disabled={pending}
          onChange={(event) =>
            onChange({ label: event.target.value || undefined })
          }
          value={variable.label ?? ""}
        />
      </Field>

      <Field hint="the line under it" label="Description">
        <TextAreaField
          disabled={pending}
          onChange={(event) =>
            onChange({ description: event.target.value || undefined })
          }
          rows={3}
          value={variable.description ?? ""}
        />
      </Field>

      {listed ? (
        <FieldGroup legend="Choices">
          {options.map((option, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: Choices stay ordered and hold no local state.
            <div className="flex flex-wrap items-center gap-2" key={index}>
              <TextField
                aria-label={`Wording for choice ${index + 1}`}
                className="min-w-32 flex-1"
                disabled={pending}
                onChange={(event) =>
                  onChange({
                    options: replaceAt(options, index, {
                      label: event.target.value,
                    }),
                  })
                }
                placeholder="Wording"
                value={option.label}
              />
              <TextField
                aria-label={`Value for choice ${index + 1}`}
                className="min-w-32 flex-1"
                disabled={pending}
                onChange={(event) =>
                  onChange({
                    options: replaceAt(options, index, {
                      value: event.target.value,
                    }),
                  })
                }
                placeholder="What reaches the prompt"
                value={option.value}
              />
              <RemoveAction
                disabled={pending}
                label={`Remove choice ${index + 1}`}
                onClick={() => onChange({ options: without(options, index) })}
              />
            </div>
          ))}
          <AddAction
            disabled={pending}
            onClick={() =>
              onChange({ options: [...options, { label: "", value: "" }] })
            }
          >
            Add choice
          </AddAction>
          {variable.widget === "multiselect" ? (
            <Field
              hint="what joins the chosen values in the prompt"
              label="Separator"
            >
              <TextField
                disabled={pending}
                onChange={(event) =>
                  onChange({ separator: event.target.value || undefined })
                }
                value={variable.separator ?? ""}
              />
            </Field>
          ) : null}
        </FieldGroup>
      ) : null}

      {numeric ? (
        <FieldGroup legend="Allowed values">
          <FieldPair>
            <Field label="Minimum">
              <TextField
                disabled={pending}
                onChange={(event) =>
                  onChange({
                    range: {
                      ...range,
                      min:
                        event.target.value === ""
                          ? undefined
                          : Number(event.target.value),
                    },
                  })
                }
                type="number"
                value={range.min ?? ""}
              />
            </Field>
            <Field label="Maximum">
              <TextField
                disabled={pending}
                onChange={(event) =>
                  onChange({
                    range: {
                      ...range,
                      max:
                        event.target.value === ""
                          ? undefined
                          : Number(event.target.value),
                    },
                  })
                }
                type="number"
                value={range.max ?? ""}
              />
            </Field>
          </FieldPair>
          <Field label="Step">
            <TextField
              disabled={pending}
              onChange={(event) =>
                onChange({
                  range: {
                    ...range,
                    step:
                      event.target.value === ""
                        ? undefined
                        : Number(event.target.value),
                  },
                })
              }
              type="number"
              value={range.step ?? ""}
            />
          </Field>
        </FieldGroup>
      ) : null}
    </div>
  );
}

export function scriptName(script: RegexScript, position: number): string {
  if (script.name?.trim()) return script.name;
  if (script.find.trim()) return script.find;
  return `Script ${position + 1}`;
}

export function ScriptListEditor({
  chosen,
  onChange,
  onChoose,
  pending,
  scripts,
}: {
  chosen: string | null;
  onChange: (scripts: RegexScript[]) => void;
  onChoose: (key: string | null) => void;
  pending: boolean;
  scripts: RegexScript[];
}) {
  return (
    <CollectionStep
      chosen={chosen}
      emptyMessage="No scripts yet. Add a script to find and replace text."
      noun="script"
      onAdd={() =>
        onChange([...scripts, { enabled: true, find: "", replace: "" }])
      }
      onChoose={onChoose}
      onMove={(from, to) => onChange(moveItem(scripts, from, to))}
      onRemove={(index) => onChange(without(scripts, index))}
      pending={pending}
      rows={scripts.map((script, index) => ({
        detail: script.find,
        id: script.id,
        name: scriptName(script, index),
        off: !script.enabled,
        search: [script.name ?? "", script.find, script.replace]
          .join(" ")
          .toLowerCase(),
      }))}
    >
      {(index) => (
        <ScriptFields
          onChange={(changes) => onChange(replaceAt(scripts, index, changes))}
          pending={pending}
          script={scripts[index]}
        />
      )}
    </CollectionStep>
  );
}

function ScriptFields({
  onChange,
  pending,
  script,
}: {
  onChange: (changes: Partial<RegexScript>) => void;
  pending: boolean;
  script: RegexScript;
}) {
  const targets = script.targets ?? [];
  const affects = script.affects ?? [];
  return (
    <div className="flex flex-col gap-6">
      <Field hint="optional" label="Name">
        <TextField
          disabled={pending}
          onChange={(event) =>
            onChange({ name: event.target.value || undefined })
          }
          value={script.name ?? ""}
        />
      </Field>

      <FieldPair>
        <Field label="Find">
          <TextField
            className="font-mono"
            disabled={pending}
            onChange={(event) => onChange({ find: event.target.value })}
            value={script.find}
          />
        </Field>
        <Field hint="g for every match, i to ignore case" label="Flags">
          <TextField
            className="font-mono"
            disabled={pending}
            onChange={(event) =>
              onChange({ flags: event.target.value || undefined })
            }
            value={script.flags ?? ""}
          />
        </Field>
      </FieldPair>

      <Field label="Replacement text">
        <TextAreaField
          disabled={pending}
          onChange={(event) => onChange({ replace: event.target.value })}
          rows={4}
          value={script.replace}
        />
      </Field>

      <FieldGroup legend="Input sources">
        {SCRIPT_TARGETS.map((target) => (
          <Switch
            checked={targets.includes(target.value)}
            key={target.value}
            label={target.label}
            onChange={(on) =>
              onChange({ targets: toggle(targets, target.value, on) })
            }
            pending={pending}
          />
        ))}
      </FieldGroup>

      <FieldGroup legend="Output targets">
        {SCRIPT_EFFECTS.map((effect) => (
          <Switch
            checked={affects.includes(effect.value)}
            key={effect.value}
            label={effect.label}
            onChange={(on) =>
              onChange({ affects: toggle(affects, effect.value, on) })
            }
            pending={pending}
          />
        ))}
      </FieldGroup>

      <FieldGroup legend="Message range">
        <FieldPair>
          <Field hint="counted from the most recent" label="Nearest message">
            <TextField
              disabled={pending}
              onChange={(event) =>
                onChange({
                  minDepth:
                    event.target.value === ""
                      ? undefined
                      : Number(event.target.value),
                })
              }
              type="number"
              value={script.minDepth ?? ""}
            />
          </Field>
          <Field label="Furthest message">
            <TextField
              disabled={pending}
              onChange={(event) =>
                onChange({
                  maxDepth:
                    event.target.value === ""
                      ? undefined
                      : Number(event.target.value),
                })
              }
              type="number"
              value={script.maxDepth ?? ""}
            />
          </Field>
        </FieldPair>
        <Switch
          checked={script.enabled}
          hint="A switched-off script stays in the preset and changes nothing."
          label="Enabled"
          onChange={(enabled) => onChange({ enabled })}
          pending={pending}
        />
        <Switch
          checked={script.runOnEdit ?? false}
          label="Run it again when a message is edited"
          onChange={(runOnEdit) => onChange({ runOnEdit })}
          pending={pending}
        />
      </FieldGroup>

      <Field hint="text cut out of the match, one per line" label="Trim">
        <TextAreaField
          disabled={pending}
          onChange={(event) =>
            onChange({ trim: readLines(event.target.value) })
          }
          rows={2}
          value={writeLines(script.trim)}
        />
      </Field>
    </div>
  );
}

function toggle<T>(values: T[], value: T, on: boolean): T[] {
  if (on) return values.includes(value) ? values : [...values, value];
  return values.filter((held) => held !== value);
}
