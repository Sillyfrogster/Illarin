"use client";

import type { KeyboardEvent, MouseEvent } from "react";
import { useLayoutEffect, useRef } from "react";
import { cn } from "@/lib/cn";

/** Where in the text the pointer landed, so a click leaves the caret there. */
function offsetAtPoint(x: number, y: number, host: HTMLElement) {
  const document_ = document as Document & {
    caretPositionFromPoint?: (x: number, y: number) => CaretPosition | null;
  };
  const position = document_.caretPositionFromPoint?.(x, y);
  if (!position || !host.contains(position.offsetNode)) return undefined;
  let counted = 0;
  const walker = document.createTreeWalker(host, NodeFilter.SHOW_TEXT);
  let node = walker.nextNode();
  while (node) {
    if (node === position.offsetNode) return counted + position.offset;
    counted += node.textContent?.length ?? 0;
    node = walker.nextNode();
  }
  return undefined;
}

function growToFit(field: HTMLTextAreaElement) {
  field.style.height = "auto";
  field.style.height = `${field.scrollHeight}px`;
}

/** One piece of page copy that is written where it sits, with no change of surface. */
export function EditableText({
  value,
  onChange,
  active,
  activate,
  done,
  id,
  live,
  label,
  placeholder,
  className,
  singleLine = false,
  as: Tag = "p",
}: {
  value: string;
  onChange: (value: string) => void;
  active: boolean;
  activate: () => void;
  done: () => void;
  id?: string;
  live: boolean;
  label: string;
  placeholder?: string;
  className?: string;
  singleLine?: boolean;
  as?: "p" | "h1" | "h2" | "h3" | "span" | "div";
}) {
  const field = useRef<HTMLTextAreaElement>(null);
  const host = useRef<HTMLHeadingElement>(null);
  const caret = useRef<number | undefined>(undefined);
  const byKeyboard = useRef(false);

  useLayoutEffect(() => {
    if (!active) {
      if (byKeyboard.current) {
        byKeyboard.current = false;
        host.current?.focus({ preventScroll: true });
      }
      return;
    }
    const input = field.current;
    if (!input) return;
    growToFit(input);
    input.focus({ preventScroll: true });
    const at = caret.current ?? input.value.length;
    input.setSelectionRange(at, at);
    caret.current = undefined;
  }, [active]);

  if (active) {
    return (
      <textarea
        aria-label={label}
        className={cn(
          "m-0 block w-full resize-none overflow-hidden border-0 bg-transparent p-0 text-ink caret-accent shadow-none outline-none! placeholder:text-mute placeholder:italic",
          className,
        )}
        id={id}
        onBlur={done}
        onChange={(event) => {
          growToFit(event.currentTarget);
          onChange(event.target.value);
        }}
        onKeyDown={(event: KeyboardEvent<HTMLTextAreaElement>) => {
          if (event.key === "Escape") {
            event.stopPropagation();
            byKeyboard.current = true;
            done();
          }
          if (singleLine && event.key === "Enter") {
            event.preventDefault();
            byKeyboard.current = true;
            done();
          }
        }}
        placeholder={placeholder}
        ref={field}
        rows={1}
        value={value}
      />
    );
  }

  const empty = !value.trim();
  return (
    <Tag
      aria-label={live ? label : undefined}
      aria-readonly={live ? true : undefined}
      className={cn(
        "m-0 whitespace-pre-wrap wrap-anywhere",
        live && "cursor-text rounded-sm focus-visible:bg-accent-wash",
        className,
        empty && "text-mute italic",
      )}
      id={id}
      onKeyDown={
        live
          ? (event: KeyboardEvent<HTMLElement>) => {
              if (event.key !== "Enter" && event.key !== " ") return;
              event.preventDefault();
              caret.current = undefined;
              activate();
            }
          : undefined
      }
      onMouseDown={
        live
          ? (event: MouseEvent<HTMLElement>) => {
              event.preventDefault();
              caret.current = offsetAtPoint(
                event.clientX,
                event.clientY,
                event.currentTarget,
              );
              activate();
            }
          : undefined
      }
      ref={host}
      role={live ? "textbox" : undefined}
      tabIndex={live ? 0 : undefined}
    >
      {value || (live ? placeholder : "")}
    </Tag>
  );
}
