import { ChipSet } from "@/components/ui/Chip";
import { RichText } from "@/components/ui/RichText";
import type {
  PresetSetting,
  PresetVariable,
  PromptListContent,
  RegexScript,
  TypedValue,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { type NamedSlot, orderSettings } from "@/lib/preset-slots";
import { themeAccent } from "@/lib/theme-colors";
import { ITEM_NAME, RUNG, STACK } from "./element-runs";

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

export function PromptList({
  content,
  isOwner,
  itemLimit,
}: {
  content: PromptListContent;
  isOwner: boolean;
  itemLimit?: number;
}) {
  const groups = content.groups ?? [];
  const fragments = (content.fragments ?? []).slice(0, itemLimit);
  const groupNames = new Map(groups.map((group) => [group.id, group.name]));
  const runs: { group?: string; fragments: PromptListContent["fragments"] }[] =
    [];
  for (const fragment of fragments) {
    const name = fragment.groupId
      ? groupNames.get(fragment.groupId)
      : undefined;
    const open = runs.at(-1);
    if (open && open.group === name) open.fragments.push(fragment);
    else runs.push({ group: name, fragments: [fragment] });
  }
  return (
    <div className="flex flex-col gap-6">
      {runs.map((run, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs hold no local state.
        <section key={index}>
          {run.group ? (
            <h4 className="mb-3 border-rule border-b pb-2 font-display text-ui font-medium text-ink">
              {run.group}
            </h4>
          ) : null}
          <Fragments
            fragments={run.fragments}
            grouped={Boolean(run.group)}
            isOwner={isOwner}
          />
        </section>
      ))}
    </div>
  );
}

function Fragments({
  fragments,
  grouped,
  isOwner,
}: {
  fragments: PromptListContent["fragments"];
  grouped: boolean;
  isOwner: boolean;
}) {
  return (
    <ol className={cn(STACK, grouped && "pl-4.5")}>
      {fragments.map((fragment, index) => (
        <li
          className={cn(RUNG, !fragment.enabled && "border-dashed opacity-60")}
          // biome-ignore lint/suspicious/noArrayIndexKey: Fragments hold no local state.
          key={index}
        >
          <div className="mb-1.5 flex flex-wrap items-baseline gap-x-2.5 gap-y-1">
            <span className="text-meta font-semibold text-ink [overflow-wrap:anywhere]">
              {fragment.name?.trim() ||
                fragment.marker?.trim() ||
                `Fragment ${index + 1}`}
            </span>
            <span className="text-meta text-mute">
              {fragment.role ? PROMPT_ROLE_LABELS[fragment.role] : null}
              {fragment.placement
                ? ` · ${PLACEMENT_LABELS[fragment.placement]}`
                : null}
              {fragment.enabled ? null : " · Off"}
              {fragment.protected ? " · Sealed prompt" : null}
            </span>
          </div>
          {fragment.protected && !isOwner ? (
            <p className="!text-ui text-mute italic">Sealed prompt</p>
          ) : fragment.marker ? (
            <p className="!text-ui text-mute italic">
              The app splices its own content in here.
            </p>
          ) : (
            <Paragraphs text={fragment.text} />
          )}
        </li>
      ))}
    </ol>
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
    <>
      {named.length > 0 ? <Settings settings={named} /> : null}
      {raw.length > 0 ? (
        <>
          <p className="mt-5 border-rule border-t pt-4 !text-label text-mute">
            As the file names them
          </p>
          <Settings raw settings={raw} />
        </>
      ) : null}
    </>
  );
}

function Settings({
  settings,
  raw,
}: {
  settings: Array<PresetSetting & { slot: NamedSlot }>;
  raw?: boolean;
}) {
  return (
    <dl
      className={cn(
        "grid items-baseline gap-x-5 gap-y-3 @max-[330px]:![grid-template-columns:minmax(0,1fr)] @max-[330px]:gap-y-1",
        raw
          ? "[grid-template-columns:fit-content(62%)_minmax(0,1fr)]"
          : "[grid-template-columns:fit-content(38%)_minmax(0,1fr)]",
      )}
    >
      {settings.map((setting) => (
        <div className="contents" key={setting.id ?? setting.name}>
          <dt
            className={cn(
              "text-mute [overflow-wrap:anywhere]",
              raw ? "font-mono text-meta" : "text-label",
            )}
          >
            {setting.slot.name}
          </dt>
          <dd className="text-ui text-ink tabular-nums [overflow-wrap:anywhere]">
            <SettingValue name={setting.name} value={setting.value} />
            {setting.slot.note ? (
              <span className="mt-0.5 block max-w-[46ch] text-meta text-mute text-pretty">
                {setting.slot.note}
              </span>
            ) : null}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/**
 * A setting's value. Text made only of spaces and newlines is shown as it is
 * written, because saying a setting holds nothing when it holds two blank
 * lines is a lie a reader would act on.
 */
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

function writeValue(value: TypedValue | undefined): string {
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

export function VariableSchema({
  variables,
  itemLimit,
}: {
  variables: PresetVariable[];
  itemLimit?: number;
}) {
  return (
    <ul className="flex list-none flex-col gap-4">
      {variables.slice(0, itemLimit).map((variable, index) => (
        <li className={RUNG} key={variable.id ?? `${index}-${variable.name}`}>
          <p className={ITEM_NAME}>{variable.label?.trim() || variable.name}</p>
          {variable.description ? (
            <RichText text={variable.description} />
          ) : null}
          {variable.options && variable.options.length > 0 ? (
            <ChipSet
              className="mt-2"
              items={variable.options.map((option, position) => ({
                id: `${position}-${option.value}`,
                label: option.label || option.value,
              }))}
              limit={KEY_PREVIEW_LIMIT}
            />
          ) : null}
        </li>
      ))}
    </ul>
  );
}

export function ScriptList({
  scripts,
  itemLimit,
}: {
  scripts: RegexScript[];
  itemLimit?: number;
}) {
  return (
    <ul className="flex list-none flex-col gap-4">
      {scripts.slice(0, itemLimit).map((script, index) => (
        <li
          className={cn(RUNG, !script.enabled && "border-dashed opacity-60")}
          key={script.id ?? index}
        >
          <p className={ITEM_NAME}>
            {script.name?.trim() || `Script ${index + 1}`}
            {script.enabled ? null : (
              <span className="font-normal text-mute"> · Off</span>
            )}
          </p>
          <p className="mt-1 flex flex-wrap items-center gap-2 [&_code]:rounded-control [&_code]:bg-deep [&_code]:px-2 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-meta [&_code]:whitespace-pre-wrap [&_code]:[overflow-wrap:anywhere]">
            <code>{script.find}</code>
            <span aria-hidden="true">→</span>
            <code>{script.replace || "nothing"}</code>
          </p>
        </li>
      ))}
    </ul>
  );
}

function Paragraphs({ text }: { text: string }) {
  const paragraphs = text.split(/\n{2,}/).filter((line) => line.trim() !== "");
  return (
    <div className="[&>p+p]:mt-[0.85em]">
      {paragraphs.map((paragraph, index) => (
        <p key={`${index}-${paragraph.slice(0, 24)}`}>{paragraph}</p>
      ))}
    </div>
  );
}
