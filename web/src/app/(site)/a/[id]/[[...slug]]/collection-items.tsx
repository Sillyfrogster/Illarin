import { RichText } from "@/components/ui/RichText";
import type { AssetElement } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import type { CollectionItem } from "@/lib/collection";
import { readEntries } from "@/lib/lorebook-entry";
import { nameSlot, orderSettings } from "@/lib/preset-slots";
import {
  ITEM_BODY,
  ITEM_META,
  ITEM_NAME,
  PASSAGE_NAME,
  PROSE,
} from "./element-runs";
import { EntryBody } from "./Lorebook";
import {
  fragmentName,
  fragmentRole,
  groupedFragments,
  PromptFragmentBody,
  ScriptBody,
  SettingBody,
  scriptName,
  VariableBody,
  variableName,
  writeValue,
} from "./PresetElements";

/** collectionItems lists whatever an element holds that answers to a name. */
export function collectionItems(
  element: AssetElement,
  isOwner: boolean,
): CollectionItem[] {
  const { content } = element;

  if (element.type === "prompt_list" && "fragments" in content) {
    return groupedFragments(content).map(({ fragment, group, place }) => ({
      detail: (
        <PromptFragmentBody
          fragment={fragment}
          isOwner={isOwner}
          place={place}
          roomy
        />
      ),
      group,
      key: fragment.id ?? `${place}`,
      name: fragmentName(fragment, place),
      note: fragmentRole(fragment),
      off: !fragment.enabled,
      terms: [fragment.marker ?? ""],
    }));
  }

  if (element.type === "entry_table" && "entries" in content) {
    return readEntries(content.entries).map((entry) => ({
      detail: <EntryBody entry={entry} />,
      key: entry.id,
      name: entry.name,
      note: entry.note,
      off: entry.isOff,
      terms: [...entry.keys, ...entry.secondaryKeys].map((key) => key.label),
    }));
  }

  if (element.type === "variable_schema" && "variables" in content) {
    return content.variables.map((variable, index) => ({
      detail: <VariableBody variable={variable} />,
      key: variable.id ?? `${index}`,
      name: variableName(variable),
      terms: [variable.name],
    }));
  }

  if (element.type === "script_list" && "scripts" in content) {
    return content.scripts.map((script, index) => ({
      detail: <ScriptBody index={index} script={script} />,
      key: script.id ?? `${index}`,
      name: scriptName(script, index),
      off: !script.enabled,
      terms: [script.find],
    }));
  }

  if (element.type === "setting_group" && "settings" in content) {
    return orderSettings(
      content.settings.filter((setting) => setting.value != null),
    ).map((setting) => ({
      detail: (
        <dl className="flex flex-col gap-1.5">
          <SettingBody
            raw={setting.slot.rank === "unrecognised"}
            setting={setting}
          />
        </dl>
      ),
      key: setting.id ?? setting.name,
      name: setting.slot.name,
      note: writeValue(setting.value),
      terms: [setting.name],
    }));
  }

  if (element.role === "extension_additions" && "fields" in content) {
    return content.fields.map((field, index) => ({
      detail: (
        <>
          <p className={ITEM_META}>{field.name}</p>
          <p className={ITEM_NAME}>{field.value}</p>
        </>
      ),
      group: field.name,
      key: `${index}`,
      name: field.value,
    }));
  }

  if (element.type === "field_list" && "fields" in content) {
    return content.fields.map((field, index) => ({
      detail: (
        <>
          <p className={ITEM_NAME}>{field.name || "Unnamed"}</p>
          <RichText className={cn(ITEM_BODY, "text-ink")} text={field.value} />
        </>
      ),
      key: `${index}`,
      name: field.name ?? "",
    }));
  }

  if (element.type === "link_list" && "links" in content) {
    return content.links.map((link, index) => ({
      detail: (
        <>
          <a
            className="font-ui text-ui font-medium text-ink underline decoration-accent/55 underline-offset-[3px] [overflow-wrap:anywhere] hover:decoration-accent"
            href={link.url}
            rel="noreferrer nofollow"
            target="_blank"
          >
            {link.label || link.url}
          </a>
          {link.note ? (
            <RichText className={ITEM_BODY} text={link.note} />
          ) : null}
        </>
      ),
      key: `${index}`,
      name: link.label || link.url,
      terms: [link.url],
    }));
  }

  if (element.type === "text_set" && "texts" in content) {
    const named = element.role === "prompt_nudges";
    return content.texts.map((item, index) => {
      const name = item.name
        ? named
          ? nameSlot(item.name).name
          : item.name
        : "";
      return {
        detail: (
          <>
            {name ? <p className={PASSAGE_NAME}>{name}</p> : null}
            <RichText className={cn(PROSE, "max-w-[70ch]")} text={item.text} />
          </>
        ),
        key: `${index}`,
        name,
        terms: item.name ? [item.name] : [],
        weight: item.text.length,
      };
    });
  }

  return [];
}
