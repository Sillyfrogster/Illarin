import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/cn";

export const controlClasses =
  "w-full min-h-11 rounded-control border-0 bg-deep px-3.5 py-2.5 font-ui text-ui text-ink transition-colors duration-200 outline-offset-2 placeholder:text-mute hover:bg-rule/40 aria-invalid:inset-ring-2 aria-invalid:inset-ring-stop motion-reduce:transition-none";

export function TextInput({ className, ...props }: ComponentProps<"input">) {
  return <input className={cn(controlClasses, className)} {...props} />;
}

export function TextArea({ className, ...props }: ComponentProps<"textarea">) {
  return (
    <textarea
      className={cn(
        controlClasses,
        "min-h-28 resize-y leading-relaxed",
        className,
      )}
      {...props}
    />
  );
}

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
  return (
    <div className={cn("grid min-w-0 gap-2", className)}>
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        {htmlFor ? (
          <label className="font-ui text-ui text-ink" htmlFor={htmlFor}>
            {label}
          </label>
        ) : (
          <span className="font-ui text-ui text-ink">{label}</span>
        )}
        {trailing}
      </div>
      {children}
      {trouble ? (
        <p
          className="font-ui text-meta text-stop"
          id={htmlFor ? `${htmlFor}-trouble` : undefined}
        >
          {trouble}
        </p>
      ) : null}
      {hint ? (
        <div
          className="font-ui text-meta text-mute"
          id={htmlFor ? `${htmlFor}-hint` : undefined}
        >
          {hint}
        </div>
      ) : null}
    </div>
  );
}

export function Trouble({ children }: { children: ReactNode }) {
  return (
    <p
      className="rounded-control bg-stop-wash px-4 py-3 font-ui text-ui text-stop"
      role="alert"
    >
      {children}
    </p>
  );
}

export function Said({
  children,
  className,
  id,
}: {
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  return (
    <output
      aria-live="polite"
      className={cn(
        "block rounded-control bg-accent-wash px-4 py-3 font-ui text-ui text-accent",
        className,
      )}
      id={id}
    >
      {children}
    </output>
  );
}
