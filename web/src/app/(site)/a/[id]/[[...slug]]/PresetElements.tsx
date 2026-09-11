import { Lock } from "lucide-react";
import { Fragment } from "react";
import { ChipSet } from "@/components/ui/Chip";
import { RichText } from "@/components/ui/RichText";
import { Run, RunHeading, RunItem } from "@/components/ui/run";
import type {
  PresetSetting,
  PresetVariable,
  PromptFragment as PromptFragmentRecord,
  PromptListContent,
  RegexScript,
  TypedValue,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { countOf } from "@/lib/collection";
import { type NamedSlot, orderSettings } from "@/lib/preset-slots";
import { themeAccent } from "@/lib/theme-colors";
import {
  ITEM_BODY,
  ITEM_META,
  ITEM_NAME,
  ITEM_VALUE,
  OFF,
  TAG,
} from "./element-runs";

const KEY_PREVIEW_LIMIT = 6;

const PROMPT_ROLE_LABELS: Record<string, string> = {
  assistant: "Assistant",
  assistant_append: "Assistant, appended",
  system: "System",
  user: "User",
  user_append: "User, appended",
};

const PLACEMENT_LABELS: Record<string, string> = {
  in_history: "In the conversation",
  post_history: "After the conversation",
  pre_history: "Before the conversation",
};

export function fragmentName(
  fragment: PromptFragmentRecord,
  place: number,
): string {
  return (
    fragment.name?.trim() || fragment.marker?.trim() || `Fragment ${place + 1}`
  );
}

export function fragmentRole(fragment: PromptFragmentRecord): string {
  return fragment.role ? (PROMPT_ROLE_LABELS[fragment.role] ?? "") : "";
}

export function fragmentNote(fragment: PromptFragmentRecord): string {
  return [
    fragmentRole(fragment),
    fragment.placement ? PLACEMENT_LABELS[fragment.placement] : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** groupedFragments keeps each fragment beside the ones its creator grouped it with. */
export function groupedFragments(content: PromptListContent) {
  const groupNames = new Map(
    (content.groups ?? []).map((group) => [group.id, group.name]),
  );
  return (content.fragments ?? []).map((fragment, place) => ({
    fragment,
    group: fragment.groupId ? groupNames.get(fragment.groupId) : undefined,
    place,
  }));
}

export function PromptList({
  content,
  isOwner,
  itemLimit,
}: {
  content: PromptListContent;
  isOwner: boolean;
  itemLimit?: number;
}) {
  const grouped = groupedFragments(content);
  const sizes = new Map<string, number>();
  for (const { group } of grouped) {
    if (group) sizes.set(group, (sizes.get(group) ?? 0) + 1);
  }
  const runs: { group?: string; fragments: typeof grouped }[] = [];
  for (const entry of grouped.slice(0, itemLimit)) {
    const open = runs.at(-1);
    if (open && open.group === entry.group) open.fragments.push(entry);
    else runs.push({ group: entry.group, fragments: [entry] });
  }
  return (
    <Run as="ol">
      {runs.map((run, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs hold no local state.
        <Fragment key={index}>
          {run.group ? (
            <RunHeading count={countOf(sizes.get(run.group) ?? 0, "fragment")}>
              {run.group}
            </RunHeading>
          ) : null}
          {run.fragments.map(({ fragment, place }) => (
            <RunItem
              className={cn(!fragment.enabled && OFF)}
              itemKey={fragment.id ?? `${place}`}
              key={fragment.id ?? place}
            >
              <PromptFragmentBody
                fragment={fragment}
                isOwner={isOwner}
                place={place}
              />
            </RunItem>
          ))}
        </Fragment>
      ))}
    </Run>
  );
}

/** PromptFragmentBody reads one fragment the same way inline and in the browser. */
export function PromptFragmentBody({
  fragment,
  isOwner,
  place,
  roomy = false,
}: {
  fragment: PromptFragmentRecord;
  isOwner: boolean;
  place: number;
  roomy?: boolean;
}) {
  const sealed = fragment.protected && !isOwner;
  const note = fragmentNote(fragment);
  return (
    <>
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
        <span className={ITEM_NAME}>{fragmentName(fragment, place)}</span>
        {sealed ? (
          <span className={cn(TAG, "bg-accent-wash text-accent")}>
            <Lock aria-hidden="true" className="size-3" />
            Sealed
          </span>
        ) : null}
        {fragment.enabled ? null : (
          <span className={cn(TAG, "bg-deep text-mute")}>Off</span>
        )}
      </div>
      {note ? <p className={ITEM_META}>{note}</p> : null}
      {sealed && roomy ? (
        <p className={cn(ITEM_BODY, "mt-1 max-w-[52ch] text-mute")}>
          The creator sealed this fragment, so its wording stays with them. The
          app still receives it in full when the preset is installed.
        </p>
      ) : null}
      {sealed ? null : fragment.marker ? (
        <p className={cn(ITEM_META, "italic")}>
          The app splices its own content in here.
        </p>
      ) : (
        <Paragraphs text={fragment.text} />
      )}
    </>
  );
}

export function SettingGroup({
  settings,
  itemLimit,
}: {
  settings: PresetSetting[];
  itemLimit?: number;
}) {
  const shown = orderSettings(
    settings.filter((setting) => setting.value != null),
  ).slice(0, itemLimit);
  if (shown.length === 0) return null;
  const named = shown.filter((setting) => setting.slot.rank !== "unrecognised");
  const raw = shown.filter((setting) => setting.slot.rank === "unrecognised");
  return (
    <Run as="dl">
      {named.map((setting) => (
        <RunItem
          as="div"
          itemKey={setting.id ?? setting.name}
          key={setting.id ?? setting.name}
        >
          <SettingBody setting={setting} />
        </RunItem>
      ))}
      {raw.length > 0 ? <RunHeading>As the file names them</RunHeading> : null}
      {raw.map((setting) => (
        <RunItem
          as="div"
          itemKey={setting.id ?? setting.name}
          key={setting.id ?? setting.name}
        >
          <SettingBody raw setting={setting} />
        </RunItem>
      ))}
    </Run>
  );
}

export function SettingBody({
  raw,
  setting,
}: {
  raw?: boolean;
  setting: PresetSetting & { slot: NamedSlot };
}) {
  return (
    <>
      <div className="flex flex-wrap items-baseline justify-between gap-x-5 gap-y-0.5">
        <dt
          className={cn(
            "min-w-0 [overflow-wrap:anywhere]",
            raw ? "font-mono text-meta text-mute" : ITEM_NAME,
          )}
        >
          {setting.slot.name}
        </dt>
        <dd className={ITEM_VALUE}>
          <SettingValue name={setting.name} value={setting.value} />
        </dd>
      </div>
      {setting.slot.note ? (
        <p className={cn(ITEM_META, "max-w-[52ch] text-pretty")}>
          {setting.slot.note}
        </p>
      ) : null}
    </>
  );
}

function SettingValue({
  name,
  value,
}: {
  name: string;
  value: TypedValue | undefined;
}) {
  const accent = name === "accent" ? themeAccent(value?.text) : null;
  if (accent) {
    return (
      <span className="inline-flex items-center gap-2">
        <span
          aria-hidden="true"
          className="size-4.5 shrink-0 rounded-full inset-ring inset-ring-rule"
          style={{ backgroundColor: accent.css }}
        />
        {accent.label}
      </span>
    );
  }
  if (value?.text != null && value.text !== "" && value.text.trim() === "") {
    return (
      <code className="block rounded-control bg-deep px-2 py-0.5 font-mono text-meta whitespace-pre-wrap">
        {value.text}
      </code>
    );
  }
  return <>{writeValue(value)}</>;
}

export function writeValue(value: TypedValue | undefined): string {
  if (!value) return "";
  if (value.number != null) {
    return value.number.toLocaleString("en-GB", { maximumFractionDigits: 20 });
  }
  if (value.boolean != null) return value.boolean ? "Yes" : "No";
  if (value.strings) {
    return value.strings.length === 0 ? "Nothing" : value.strings.join(", ");
  }
  return value.text === "" ? "Nothing" : (value.text ?? "");
}

export function variableName(variable: PresetVariable): string {
  return variable.label?.trim() || variable.name;
}

export function VariableSchema({
  variables,
  itemLimit,
}: {
  variables: PresetVariable[];
  itemLimit?: number;
}) {
  return (
    <Run>
      {variables.slice(0, itemLimit).map((variable, index) => (
        <RunItem
          itemKey={variable.id ?? `${index}`}
          key={variable.id ?? `${index}-${variable.name}`}
        >
          <VariableBody variable={variable} />
        </RunItem>
      ))}
    </Run>
  );
}

export function VariableBody({ variable }: { variable: PresetVariable }) {
  return (
    <>
      <p className={ITEM_NAME}>{variableName(variable)}</p>
      {variable.description ? (
        <RichText className={ITEM_BODY} text={variable.description} />
      ) : null}
      {variable.options && variable.options.length > 0 ? (
        <ChipSet
          className="mt-1"
          items={variable.options.map((option, position) => ({
            id: `${position}-${option.value}`,
            label: option.label || option.value,
          }))}
          limit={KEY_PREVIEW_LIMIT}
        />
      ) : null}
    </>
  );
}

export function scriptName(script: RegexScript, index: number): string {
  return script.name?.trim() || `Script ${index + 1}`;
}

export function ScriptList({
  scripts,
  itemLimit,
}: {
  scripts: RegexScript[];
  itemLimit?: number;
}) {
  return (
    <Run>
      {scripts.slice(0, itemLimit).map((script, index) => (
        <RunItem
          className={cn(!script.enabled && OFF)}
          itemKey={script.id ?? `${index}`}
          key={script.id ?? index}
        >
          <ScriptBody index={index} script={script} />
        </RunItem>
      ))}
    </Run>
  );
}

export function ScriptBody({
  index,
  script,
}: {
  index: number;
  script: RegexScript;
}) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
        <span className={ITEM_NAME}>{scriptName(script, index)}</span>
        {script.enabled ? null : (
          <span className={cn(TAG, "bg-deep text-mute")}>Off</span>
        )}
      </div>
      <p className="flex flex-wrap items-center gap-2 font-ui text-meta text-mute [&_code]:rounded-control [&_code]:bg-deep [&_code]:px-2 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-meta [&_code]:text-ink [&_code]:whitespace-pre-wrap [&_code]:[overflow-wrap:anywhere]">
        <code>{script.find}</code>
        <span aria-hidden="true">→</span>
        <code>{script.replace || "nothing"}</code>
      </p>
    </>
  );
}

function Paragraphs({ text }: { text: string }) {
  const paragraphs = text.split(/\n{2,}/).filter((line) => line.trim() !== "");
  return (
    <div className={cn(ITEM_BODY, "max-w-[70ch] [&>p+p]:mt-[0.85em]")}>
      {paragraphs.map((paragraph, index) => (
        <p key={`${index}-${paragraph.slice(0, 24)}`}>{paragraph}</p>
      ))}
    </div>
  );
}
