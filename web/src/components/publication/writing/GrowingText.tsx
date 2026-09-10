"use client";

import { useEffect, useRef } from "react";

/** A field as tall as what has been typed into it, so a title reads as a title. */
export function GrowingText({
  className,
  id,
  maxLength,
  onChange,
  placeholder,
  shown = true,
  value,
}: {
  className: string;
  id: string;
  maxLength: number;
  onChange: (value: string) => void;
  placeholder?: string;
  /** A hidden field measures nothing, so it is measured again on the way back. */
  shown?: boolean;
  value: string;
}) {
  const field = useRef<HTMLTextAreaElement>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: the height follows the text, which biome cannot see through scrollHeight.
  useEffect(() => {
    const element = field.current;
    if (!element || !shown) return;
    element.style.height = "auto";
    element.style.height = `${element.scrollHeight}px`;
  }, [value, shown]);

  return (
    <textarea
      className={className}
      id={id}
      maxLength={maxLength}
      onChange={(event) => onChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Enter") event.preventDefault();
      }}
      placeholder={placeholder}
      ref={field}
      rows={1}
      value={value}
    />
  );
}
