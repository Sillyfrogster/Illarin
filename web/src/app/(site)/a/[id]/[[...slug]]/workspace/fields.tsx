"use client";

import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { Select } from "@/components/ui/select";
import { cn } from "@/lib/cn";

const MOVE =
  "inline-flex min-h-11 items-center gap-1.5 rounded-control px-3 text-meta font-medium text-mute outline-offset-3 hover:bg-deep hover:text-ink disabled:opacity-40 disabled:hover:bg-transparent";

const CONTROL =
  "w-full rounded-control border-0 bg-deep px-3 text-ui text-ink outline-offset-3 placeholder:text-mute disabled:opacity-60";

export function Field({
  children,
  hint,
  label,
}: {
  children: ReactNode;
  hint?: string;
  label: string;
}) {
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: The control is the child, and the rule cannot see through the boundary.
    <label className="block">
      <span className="mb-2 block text-label font-medium text-mute">
        {label}
        {hint ? <span className="font-normal"> · {hint}</span> : null}
      </span>
      {children}
    </label>
  );
}

export function FieldGroup({
  children,
  legend,
}: {
  children: ReactNode;
  legend: string;
}) {
  return (
    <fieldset className="min-w-0 border-0 p-0">
      <legend className="mb-3 font-display text-ui font-medium text-ink">
        {legend}
      </legend>
      <div className="flex flex-col gap-4">{children}</div>
    </fieldset>
  );
}

export function FieldPair({ children }: { children: ReactNode }) {
  return <div className="grid gap-4 @sm:grid-cols-2">{children}</div>;
}

export function TextField(props: ComponentProps<"input">) {
  return (
    <input
      {...props}
      className={cn(CONTROL, "min-h-11 py-2", props.className)}
    />
  );
}

export function TextAreaField(props: ComponentProps<"textarea">) {
  return (
    <textarea
      {...props}
      className={cn(CONTROL, "resize-y py-3 leading-relaxed", props.className)}
    />
  );
}

export function ChoiceField(props: ComponentProps<"select">) {
  return <Select {...props} className={cn("w-full", props.className)} />;
}

export function Switch({
  checked,
  hint,
  label,
  onChange,
  pending,
}: {
  checked: boolean;
  hint?: string;
  label: string;
  onChange: (checked: boolean) => void;
  pending: boolean;
}) {
  return (
    <label
      className={cn(
        "flex min-h-11 cursor-pointer items-start gap-3 rounded-control p-3 text-ui text-ink",
        checked ? "bg-accent-wash" : "bg-deep",
        pending && "opacity-60",
      )}
    >
      <input
        checked={checked}
        className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
        disabled={pending}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span className="min-w-0">
        {label}
        {hint ? (
          <span className="mt-1 block text-meta text-mute">{hint}</span>
        ) : null}
      </span>
    </label>
  );
}

export function AddAction({
  children,
  disabled,
  onClick,
}: {
  children: ReactNode;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      className="inline-flex min-h-11 items-center gap-2 self-start rounded-control bg-deep px-4 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45"
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      <Plus aria-hidden="true" size={16} />
      {children}
    </button>
  );
}

export function RemoveAction({
  children,
  disabled,
  label,
  onClick,
}: {
  children?: ReactNode;
  disabled?: boolean;
  label?: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-label={label}
      className={cn(
        "inline-flex min-h-11 items-center gap-2 rounded-control text-meta font-medium text-stop outline-offset-3 hover:bg-stop-wash disabled:opacity-45",
        children ? "px-3" : "size-11 justify-center",
      )}
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      <Trash2 aria-hidden="true" size={15} />
      {children}
    </button>
  );
}

export function Note({ children }: { children: ReactNode }) {
  return <p className="text-meta text-mute">{children}</p>;
}

export function InlineItem({
  children,
  moves,
  name,
  onRemove,
  pending,
  removeLabel,
}: {
  children: ReactNode;
  moves?: ItemMoves;
  name: string;
  onRemove: () => void;
  pending: boolean;
  removeLabel: string;
}) {
  return (
    <div className="flex flex-col gap-3">
      <p className="font-display text-ui font-medium text-ink">{name}</p>
      {children}
      <div className="flex flex-wrap items-center gap-1">
        {moves ? <ItemMoveActions moves={moves} pending={pending} /> : null}
        <RemoveAction disabled={pending} onClick={onRemove}>
          {removeLabel}
        </RemoveAction>
      </div>
    </div>
  );
}

export type ItemMoves = {
  onEarlier: () => void;
  onLater: () => void;
  position: number;
  total: number;
};

export function ItemMoveActions({
  moves,
  pending,
}: {
  moves: ItemMoves;
  pending: boolean;
}) {
  return (
    <>
      <button
        className={MOVE}
        disabled={pending || moves.position === 0}
        onClick={moves.onEarlier}
        type="button"
      >
        <ArrowUp aria-hidden="true" size={14} />
        Earlier
      </button>
      <button
        className={MOVE}
        disabled={pending || moves.position === moves.total - 1}
        onClick={moves.onLater}
        type="button"
      >
        <ArrowDown aria-hidden="true" size={14} />
        Later
      </button>
    </>
  );
}
