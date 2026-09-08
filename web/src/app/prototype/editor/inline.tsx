"use client";

import type { KeyboardEvent, MouseEvent } from "react";
import { useLayoutEffect, useRef } from "react";
import { cn } from "./ui";

/** Where in the text the pointer landed, so a click leaves the caret there */
function offsetAtPoint(x: number, y: number, host: HTMLElement) {
  const doc = document as Document & {
    caretPositionFromPoint?: (x: number, y: number) => CaretPosition | null;
  };
  const position = doc.caretPositionFromPoint?.(x, y);
  if (!position || !host.contains(position.offsetNode)) return undefined;
  let count = 0;
  const walker = document.createTreeWalker(host, NodeFilter.SHOW_TEXT);
  let node = walker.nextNode();
  while (node) {
    if (node === position.offsetNode) return count + position.offset;
    count += node.textContent?.length ?? 0;
    node = walker.nextNode();
  }
  return undefined;
}

/** One piece of page copy that edits where it sits, with no change of surface */
export function Inline({
  value,
  onChange,
  active,
  activate,
  done,
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
  live: boolean;
  label: string;
  placeholder?: string;
  className?: string;
  singleLine?: boolean;
  as?: "p" | "h1" | "h4" | "span" | "div";
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
    input.focus({ preventScroll: true });
    const at = caret.current ?? input.value.length;
    input.setSelectionRange(at, at);
    caret.current = undefined;
  }, [active]);

  if (active)
    return (
      <textarea
        ref={field}
        aria-label={label}
        value={value}
        rows={1}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        onBlur={done}
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
        className={cn(
          "w-field w-caret ws:m-0 ws:block ws:w-full ws:resize-none ws:overflow-hidden ws:border-0 ws:bg-transparent ws:p-0 ws:text-ink ws:shadow-none ws:outline-none ws:placeholder:text-mute ws:placeholder:italic",
          className,
        )}
      />
    );

  const empty = !value.trim();
  return (
    <Tag
      ref={host}
      role={live ? "textbox" : undefined}
      aria-readonly={live ? true : undefined}
      aria-label={live ? label : undefined}
      tabIndex={live ? 0 : undefined}
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
      onKeyDown={
        live
          ? (event: KeyboardEvent<HTMLElement>) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                caret.current = undefined;
                activate();
              }
            }
          : undefined
      }
      className={cn(
        "ws:m-0 ws:whitespace-pre-wrap ws:wrap-anywhere",
        live && "w-editable ws:cursor-text",
        empty && "ws:text-mute ws:italic",
        className,
      )}
    >
      {value || (live ? placeholder : "")}
    </Tag>
  );
}
