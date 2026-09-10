import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { cn } from "@/lib/cn";

/** How a thing is doing, said in colour as well as words. */
export type Tone = "quiet" | "accent" | "stop";

const TONES: Record<Tone, string> = {
  accent: "bg-accent-wash text-accent",
  quiet: "bg-deep text-mute group-hover:bg-rule/45",
  stop: "bg-stop-wash text-stop",
};

/** The head of a register, with the one thing you can start from here. */
export function PanelHead({
  action,
  id,
  title,
}: {
  action?: ReactNode;
  id: string;
  title: string;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4">
      <h2
        className="font-display text-section font-medium tracking-tight text-ink"
        id={id}
      >
        {title}
      </h2>
      {action}
    </div>
  );
}

/** Everything a register holds, as rows on the page rather than cells in a table. */
export function Rows({ children }: { children: ReactNode }) {
  return (
    <ul className="-mx-4 mt-4 flex list-none flex-col sm:-mx-5">{children}</ul>
  );
}

/** One row; its name opens it and covers the row, so other controls sit above. */
export function Row({
  aside,
  children,
  facts,
  lead,
  onOpen,
  open,
  standing,
  title,
  trailing,
}: {
  /** Controls at the end of the row, which stand above the name stretched under them. */
  aside?: ReactNode;
  /** Anything below the facts, such as a control or a disclosure. */
  children?: ReactNode;
  facts?: ReactNode;
  /** A portrait, mark or state light before the name. */
  lead?: ReactNode;
  onOpen?: () => void;
  /** What opening this row is called, for anyone who cannot see the row. */
  open?: string;
  standing?: ReactNode;
  title: ReactNode;
  /** A mark beside the name, such as what state this is in. */
  trailing?: ReactNode;
}) {
  return (
    <li className="group relative flex min-w-0 gap-4 rounded-plate px-4 py-5 transition-colors duration-200 not-first:border-t not-first:border-rule/45 hover:border-transparent hover:bg-deep motion-reduce:transition-none sm:px-5">
      {lead}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-2">
          <h3 className="min-w-0 font-display text-section leading-snug font-medium text-ink wrap-anywhere">
            {onOpen ? (
              <button
                className="text-left text-ink outline-offset-3 before:absolute before:inset-0 before:content-[''] hover:text-accent"
                onClick={onOpen}
                type="button"
              >
                {title}
                {open ? <span className="sr-only">. {open}</span> : null}
              </button>
            ) : (
              title
            )}
          </h3>
          {trailing}
        </div>
        {facts ? (
          <p className="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 font-prose text-meta text-mute">
            {facts}
          </p>
        ) : null}
        {standing ? (
          <p className="mt-2 max-w-[62ch] font-prose text-meta text-mute">
            {standing}
          </p>
        ) : null}
        {children}
      </div>
      {aside ? (
        <div className="relative flex shrink-0 items-start gap-1">{aside}</div>
      ) : null}
    </li>
  );
}

/** A plate carrying a picture or a mark, at the size a row's name sits against. */
export function RowMark({
  children,
  tone = "quiet",
}: {
  children: ReactNode;
  tone?: Tone;
}) {
  return (
    <span
      className={cn(
        "mt-0.5 grid size-11 shrink-0 place-items-center overflow-hidden rounded-control",
        tone === "quiet" ? "bg-deep text-mute group-hover:bg-rule/45" : "",
        tone === "accent" ? "bg-accent-wash text-accent" : "",
        tone === "stop" ? "bg-stop-wash text-stop" : "",
      )}
    >
      {children}
    </span>
  );
}

/** A short standing worn beside a name. */
export function Mark({
  children,
  icon: Icon,
  tone = "quiet",
}: {
  children: ReactNode;
  icon?: LucideIcon;
  tone?: Tone;
}) {
  return (
    <span
      className={cn(
        "inline-flex min-h-7 items-center gap-1.5 rounded-control px-2.5 font-ui text-label font-medium whitespace-nowrap",
        TONES[tone],
      )}
    >
      {Icon ? (
        <Icon aria-hidden="true" className="size-3.5" strokeWidth={2} />
      ) : null}
      {children}
    </span>
  );
}

/** A way to move a row within an order a reader will see. */
export function RowMove({
  disabled,
  icon: Icon,
  label,
  onClick,
}: {
  disabled?: boolean;
  icon: LucideIcon;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-label={label}
      className="inline-flex size-11 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-plane hover:text-ink disabled:opacity-30 disabled:hover:bg-transparent"
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      <Icon aria-hidden="true" className="size-4" strokeWidth={1.9} />
    </button>
  );
}

/** A control on a row, which has to stand above the name stretched under it. */
export function RowAction({
  busy,
  children,
  onClick,
}: {
  busy?: boolean;
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      className="relative inline-flex min-h-11 items-center rounded-control bg-deep px-4 font-ui text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45 group-hover:bg-plane"
      disabled={busy}
      onClick={onClick}
      type="button"
    >
      {children}
    </button>
  );
}

/** What to do about an empty register, rather than the news that it is empty. */
export function Nothing({ children }: { children: ReactNode }) {
  return (
    <p className="mt-8 max-w-[54ch] font-prose text-prose text-mute">
      {children}
    </p>
  );
}

/** What a register no longer uses, kept reachable without being in the way. */
export function Past({
  children,
  summary,
}: {
  children: ReactNode;
  summary: string;
}) {
  return (
    <div className="mt-8 max-w-[34rem]">
      <MorphingDisclosure
        className="rounded-plate bg-deep px-5 py-4"
        summary={summary}
      >
        <ul className="mt-4 flex list-none flex-col gap-1">{children}</ul>
      </MorphingDisclosure>
    </div>
  );
}

/** One line of what a register no longer uses, and the way to bring it back. */
export function PastRow({
  action,
  children,
}: {
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <li className="flex flex-wrap items-center justify-between gap-3 border-rule/45 py-1 font-prose text-meta text-mute not-first:border-t">
      <span className="min-w-0 wrap-anywhere">{children}</span>
      {action}
    </li>
  );
}

/** The way a register starts something new, at the head of the register itself. */
export function StartAction({
  children,
  disabled,
  icon: Icon,
  onClick,
}: {
  children: ReactNode;
  disabled?: boolean;
  icon: LucideIcon;
  onClick: () => void;
}) {
  return (
    <button
      className="inline-flex min-h-11 shrink-0 items-center gap-2 rounded-control bg-action px-5 font-ui text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90 disabled:opacity-40"
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      <Icon aria-hidden="true" className="size-4" strokeWidth={2} />
      {children}
    </button>
  );
}
