"use client";

import {
  type AriaAttributes,
  createContext,
  type ReactNode,
  useContext,
  useId,
} from "react";
import { cn } from "@/lib/cn";

type FieldIds = { id: string; describedBy?: string; invalid: boolean };

const FieldContext = createContext<FieldIds | null>(null);

/** useFieldControl gives a control inside a Field the field's id and ties its hint and error to it, unless the control sets its own. */
export function useFieldControl(props: {
  id?: string;
  "aria-describedby"?: string;
  "aria-invalid"?: AriaAttributes["aria-invalid"];
}) {
  const field = useContext(FieldContext);
  if (!field) return {};
  return {
    id: props.id ?? field.id,
    "aria-describedby": props["aria-describedby"] ?? field.describedBy,
    "aria-invalid": props["aria-invalid"] ?? (field.invalid || undefined),
  };
}

/** Field is shadcn's field: a label, the control, an error and a hint, with the hint and error wired to the control for screen readers. */
export function Field({
  children,
  className,
  hint,
  htmlFor,
  label,
  trailing,
  trouble,
}: {
  children: ReactNode;
  className?: string;
  hint?: ReactNode;
  htmlFor?: string;
  label: ReactNode;
  trailing?: ReactNode;
  trouble?: string;
}) {
  const generated = useId();
  const id = htmlFor ?? generated;
  const describedBy =
    [trouble ? `${id}-trouble` : null, hint ? `${id}-hint` : null]
      .filter(Boolean)
      .join(" ") || undefined;
  return (
    <FieldContext.Provider value={{ id, describedBy, invalid: !!trouble }}>
      <div
        className={cn("grid min-w-0 gap-2", className)}
        data-invalid={trouble ? "" : undefined}
      >
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <label className="font-ui text-ui font-medium text-ink" htmlFor={id}>
            {label}
          </label>
          {trailing}
        </div>
        {children}
        {trouble ? (
          <p
            className="font-ui text-meta text-stop"
            id={`${id}-trouble`}
            role="alert"
          >
            {trouble}
          </p>
        ) : null}
        {hint ? (
          <div className="font-ui text-meta text-mute" id={`${id}-hint`}>
            {hint}
          </div>
        ) : null}
      </div>
    </FieldContext.Provider>
  );
}
