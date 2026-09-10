"use client";

import { controlClasses, Field } from "@/components/ui/field";
import { cn } from "@/lib/cn";
import { type LocalParts, zoneLabel } from "@/lib/schedule-time";

/** When an edition goes live, in the zone the writer's own browser is in. */
export function ScheduleFields({
  disabled,
  id,
  parts,
  onChange,
}: {
  disabled?: boolean;
  id: string;
  parts: LocalParts;
  onChange: (parts: LocalParts) => void;
}) {
  return (
    <Field
      hint={`Times are ${zoneLabel()}, the zone this browser is in.`}
      htmlFor={`${id}-date`}
      label="Goes live"
    >
      <div className="flex flex-wrap gap-2">
        <input
          className={cn(controlClasses, "w-auto flex-[1_1_10rem]")}
          disabled={disabled}
          id={`${id}-date`}
          onChange={(event) => onChange({ ...parts, date: event.target.value })}
          type="date"
          value={parts.date}
        />
        <input
          aria-label="Time it goes live"
          className={cn(controlClasses, "w-auto flex-[1_1_7rem]")}
          disabled={disabled}
          id={`${id}-time`}
          onChange={(event) => onChange({ ...parts, time: event.target.value })}
          type="time"
          value={parts.time}
        />
      </div>
    </Field>
  );
}
