import { ChipSet } from "@/components/ui/Chip";
import { RichText } from "@/components/ui/RichText";
import { Run, RunItem } from "@/components/ui/run";
import type { LorebookEntry } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { type EntryPresentation, readEntries } from "@/lib/lorebook-entry";
import { ITEM_BODY, ITEM_META, ITEM_NAME, OFF, TAG } from "./element-runs";

const KEY_PREVIEW_LIMIT = 6;

export function LorebookEntries({
  entries,
  itemLimit,
}: {
  entries: LorebookEntry[];
  itemLimit?: number;
}) {
  return (
    <Run as="ol">
      {readEntries(entries)
        .slice(0, itemLimit)
        .map((entry) => (
          <RunItem
            className={cn(entry.isOff && OFF)}
            itemKey={entry.id}
            key={entry.id}
          >
            <EntryBody entry={entry} />
          </RunItem>
        ))}
    </Run>
  );
}

/** EntryBody reads one entry the same way inline and in the browser. */
export function EntryBody({ entry }: { entry: EntryPresentation }) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
        <span className={ITEM_NAME}>{entry.name}</span>
        {entry.isOff ? (
          <span className={cn(TAG, "bg-deep text-mute")}>Off</span>
        ) : null}
      </div>
      <p className={ITEM_META}>{entry.firing.join(" · ")}</p>
      {entry.keys.length > 0 ? (
        <ChipSet
          className="mt-0.5"
          items={entry.keys}
          limit={KEY_PREVIEW_LIMIT}
        />
      ) : null}
      {entry.secondaryKeys.length > 0 ? (
        <div className="flex flex-wrap items-baseline gap-x-2.5 gap-y-1.5">
          <p className={ITEM_META}>Second keys</p>
          <ChipSet items={entry.secondaryKeys} limit={KEY_PREVIEW_LIMIT} />
        </div>
      ) : null}
      {entry.text.trim() ? (
        <RichText
          className={cn(ITEM_BODY, "mt-1 max-w-[70ch]")}
          text={entry.text}
        />
      ) : null}
    </>
  );
}
