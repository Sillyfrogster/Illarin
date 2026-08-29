"use client";

import { useEffect, useRef } from "react";

export function GrowingText({
  className,
  id,
  maxLength,
  onChange,
  placeholder,
  value,
}: {
  className: string;
  id: string;
  maxLength: number;
  onChange: (value: string) => void;
  placeholder?: string;
  value: string;
}) {
  const field = useRef<HTMLTextAreaElement>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: the height follows the text, which biome cannot see through scrollHeight.
  useEffect(() => {
    const element = field.current;
    if (!element) return;
    element.style.height = "auto";
    element.style.height = `${element.scrollHeight}px`;
  }, [value]);

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
