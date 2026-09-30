"use client";

import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";

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
    <Button className="self-start" disabled={disabled} onClick={onClick}>
      <Plus aria-hidden="true" />
      {children}
    </Button>
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
  const button = (
    <Button
      aria-label={label}
      className="text-stop hover:text-stop"
      disabled={disabled}
      onClick={onClick}
      size={children ? "default" : "icon"}
      variant="ghost"
    >
      <Trash2 aria-hidden="true" />
      {children}
    </Button>
  );
  return label && !children ? (
    <Tooltip content={label}>{button}</Tooltip>
  ) : (
    button
  );
}

export function Note({ children }: { children: ReactNode }) {
  return <p className="text-meta text-mute">{children}</p>;
}

export function InlineItem({
  children,
  handle,
  name,
  onRemove,
  pending,
  removeLabel,
}: {
  children: ReactNode;
  handle?: ReactNode;
  name: string;
  onRemove: () => void;
  pending: boolean;
  removeLabel: string;
}) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-1">
        {handle}
        <p className="font-display text-ui font-medium text-ink">{name}</p>
      </div>
      {children}
      <div className="flex flex-wrap items-center gap-1">
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
      <Button
        disabled={pending || moves.position === 0}
        onClick={moves.onEarlier}
        variant="ghost"
      >
        <ArrowUp aria-hidden="true" />
        Earlier
      </Button>
      <Button
        disabled={pending || moves.position === moves.total - 1}
        onClick={moves.onLater}
        variant="ghost"
      >
        <ArrowDown aria-hidden="true" />
        Later
      </Button>
    </>
  );
}
