"use client";

import { Field } from "@/components/console/Field";
import { type LocalParts, zoneLabel } from "@/lib/schedule-time";
import styles from "./ScheduleFields.module.css";

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
      <div className={styles.pair}>
        <input
          disabled={disabled}
          id={`${id}-date`}
          onChange={(event) => onChange({ ...parts, date: event.target.value })}
          type="date"
          value={parts.date}
        />
        <input
          aria-label="Time it goes live"
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
