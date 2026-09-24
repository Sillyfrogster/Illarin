"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { AlertCircle, Check, FileUp, Upload } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type DragEvent, useEffect, useId, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { readUploadOperation, type UploadOperation } from "@/lib/api/query";
import type { BuildChoices } from "@/lib/api/shapes";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { fileWeight } from "@/lib/file-weight";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { StartFromNothing } from "./StartFromNothing";
import { sendFile } from "./send-file";

const POLL_MS = 600;

const LOST =
  "The connection dropped. Your file is safe. Check again to see where it got to.";

type Phase =
  | { at: "choosing" }
  | { at: "sending"; file: File; sent: number }
  | { at: "reading"; file: File; operation: UploadOperation }
  | { at: "lost"; file: File; operation: UploadOperation }
  | { at: "opening"; file: File; label: string }
  | { at: "refused"; file: File; message: string };

type Busy = Exclude<Phase, { at: "choosing" }>;

const STEPS = ["Upload", "Read", "Open the draft"] as const;

/** UploadFlow sends a chosen file straight away and opens the draft it becomes. */
export function UploadFlow({ choices }: { choices: BuildChoices | null }) {
  const { account } = useAuth();
  const router = useRouter();
  const [phase, setPhase] = useState<Phase>({ at: "choosing" });

  const reading = phase.at === "reading" ? phase : null;
  useEffect(() => {
    if (!reading) return;
    let active = true;
    const timer = setTimeout(async () => {
      try {
        const next = await readUploadOperation(reading.operation.url);
        if (!active) return;
        if (next.status === "failed") {
          setPhase({
            at: "refused",
            file: reading.file,
            message:
              next.failure?.message ?? "Illarin could not read this file.",
          });
        } else if (next.status === "success" && next.work) {
          const label = TYPE_LABELS[next.work.type as keyof typeof TYPE_LABELS];
          setPhase({
            at: "opening",
            file: reading.file,
            label: label?.toLowerCase() ?? "draft",
          });
          router.push(workHref(next.work.id, next.work.name));
        } else {
          setPhase({ ...reading, operation: next });
        }
      } catch {
        if (active) setPhase({ ...reading, at: "lost" });
      }
    }, POLL_MS);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [reading, router]);

  async function send(file: File) {
    setPhase({ at: "sending", file, sent: 0 });
    const sent = await sendFile(file, (fraction) =>
      setPhase((current) =>
        current.at === "sending" ? { ...current, sent: fraction } : current,
      ),
    );
    setPhase(
      sent.operation
        ? { at: "reading", file, operation: sent.operation }
        : { at: "refused", file, message: sent.error },
    );
  }

  if (account === undefined) {
    return (
      <output className="mt-8 block text-ui text-mute">
        Loading your account…
      </output>
    );
  }

  if (!account) {
    return (
      <Gate
        action="Sign in"
        href="/sign-in"
        heading="Sign in before you publish"
      >
        Sign in to import a file or create a draft.
      </Gate>
    );
  }

  if (!account.emailVerified) {
    return (
      <Gate
        action="Verify email"
        href="/verify-email?returnTo=%2Fupload"
        heading="Verify your email before you publish"
      >
        Verify your email to import a file or create a draft.
      </Gate>
    );
  }

  return (
    <div className="mt-8">
      <AnimatePresence initial={false} mode="wait">
        {phase.at === "choosing" ? (
          <DropFile key="drop" onFile={send} />
        ) : (
          <FileProgress
            key="progress"
            onBeginAgain={() => setPhase({ at: "choosing" })}
            onCheckAgain={() =>
              phase.at === "lost" && setPhase({ ...phase, at: "reading" })
            }
            phase={phase}
          />
        )}
      </AnimatePresence>
      {phase.at === "choosing" || phase.at === "refused" ? (
        <StartFromNothing choices={choices} />
      ) : null}
    </div>
  );
}

const SWAP = {
  initial: { opacity: 0, scale: 0.97 },
  animate: { opacity: 1, scale: 1 },
  exit: { opacity: 0, scale: 0.97 },
  transition: { duration: 0.22, ease: [0.22, 1, 0.36, 1] },
} as const;

function DropFile({ onFile }: { onFile: (file: File) => void }) {
  const field = useId();
  const [over, setOver] = useState(false);
  const still = useReducedMotion();

  function take(file: File | undefined) {
    setOver(false);
    if (file) onFile(file);
  }

  function drop(event: DragEvent<HTMLLabelElement>) {
    event.preventDefault();
    take(event.dataTransfer.files?.[0]);
  }

  return (
    <motion.div {...(still ? {} : SWAP)}>
      <input
        className="peer sr-only"
        id={field}
        onChange={(event) => take(event.target.files?.[0])}
        type="file"
      />
      <label
        className={cn(
          "group flex min-h-64 cursor-pointer flex-col items-center justify-center gap-3 rounded-plate border-2 border-dashed px-6 py-10 text-center transition-[border-color,background-color,transform] duration-200 motion-reduce:transition-none",
          "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-accent peer-focus-visible:outline-offset-3",
          over
            ? "scale-[1.01] border-accent bg-accent-wash"
            : "border-edge bg-inset hover:border-accent hover:bg-accent-wash",
        )}
        htmlFor={field}
        onDragLeave={() => setOver(false)}
        onDragOver={(event) => {
          event.preventDefault();
          setOver(true);
        }}
        onDrop={drop}
      >
        <span
          className={cn(
            "flex size-14 items-center justify-center rounded-full bg-plane text-accent shadow-sm transition-transform duration-300 ease-[var(--ease-wipe)] group-hover:-translate-y-1 motion-reduce:transform-none",
            over && "-translate-y-1.5",
          )}
        >
          <Upload aria-hidden="true" size={24} strokeWidth={1.6} />
        </span>
        <span className="text-lede font-medium text-ink">
          {over ? "Drop to upload" : "Drop a file here, or choose one"}
        </span>
        <span className="max-w-[44ch] text-meta text-mute">
          A character card, lorebook, preset, theme, pack or extension zip. It
          uploads as soon as you choose it.
        </span>
      </label>
    </motion.div>
  );
}

function FileProgress({
  onBeginAgain,
  onCheckAgain,
  phase,
}: {
  onBeginAgain: () => void;
  onCheckAgain: () => void;
  phase: Busy;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  const still = useReducedMotion();
  const refused = phase.at === "refused";
  const step = { sending: 0, reading: 1, lost: 1, opening: 2, refused: -1 }[
    phase.at
  ];

  useEffect(() => {
    if (refused) heading.current?.focus();
  }, [refused]);

  const title = {
    sending: `Uploading ${Math.round((phase.at === "sending" ? phase.sent : 0) * 100)}%`,
    reading: "Reading your file",
    lost: "Import status unavailable",
    opening: `Opening your ${phase.at === "opening" ? phase.label : "draft"}`,
    refused: "This file was not added",
  }[phase.at];

  const filled =
    phase.at === "sending"
      ? phase.sent * 0.5
      : phase.at === "opening"
        ? 1
        : 0.5;

  return (
    <motion.section
      {...(still ? {} : SWAP)}
      aria-live="polite"
      className={cn(
        "rounded-plate p-6 sm:p-8",
        refused ? "bg-stop-wash" : "bg-inset",
      )}
    >
      <div className="flex items-center gap-4">
        <span
          className={cn(
            "flex size-12 shrink-0 items-center justify-center rounded-control bg-plane",
            refused ? "text-stop" : "text-accent",
          )}
        >
          {refused ? (
            <AlertCircle aria-hidden="true" size={22} strokeWidth={1.6} />
          ) : (
            <FileUp aria-hidden="true" size={22} strokeWidth={1.6} />
          )}
        </span>
        <div className="min-w-0">
          <p className="truncate text-ui font-medium text-ink">
            {phase.file.name}
          </p>
          <p className="text-meta text-mute">{fileWeight(phase.file.size)}</p>
        </div>
      </div>

      <h2
        className="mt-6 font-display text-section font-medium text-ink tabular-nums outline-none"
        ref={heading}
        tabIndex={-1}
      >
        {title}
      </h2>

      {refused ? null : (
        <>
          <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-deep">
            <motion.div
              animate={{ width: `${filled * 100}%` }}
              className="relative h-full overflow-hidden rounded-full bg-action"
              initial={false}
              transition={{
                duration: still ? 0 : 0.4,
                ease: [0.22, 1, 0.36, 1],
              }}
            >
              {phase.at === "reading" ? (
                <span className="absolute inset-y-0 w-2/5 bg-white/35 motion-safe:animate-sweep motion-reduce:hidden" />
              ) : null}
            </motion.div>
          </div>
          <ol className="mt-5 flex list-none flex-wrap gap-x-6 gap-y-2 p-0">
            {STEPS.map((name, index) => (
              <li
                className={cn(
                  "flex items-center gap-2 text-meta transition-colors duration-300",
                  index <= step ? "text-ink" : "text-mute",
                )}
                key={name}
              >
                <span
                  className={cn(
                    "flex size-5 items-center justify-center rounded-full transition-colors duration-300",
                    index < step
                      ? "bg-action text-white"
                      : index === step
                        ? "bg-accent-wash text-accent"
                        : "bg-deep",
                  )}
                >
                  {index < step ? (
                    <motion.span
                      animate={{ scale: 1 }}
                      initial={still ? false : { scale: 0 }}
                      transition={{
                        type: "spring",
                        stiffness: 500,
                        damping: 24,
                      }}
                    >
                      <Check aria-hidden="true" size={12} strokeWidth={3} />
                    </motion.span>
                  ) : index === step ? (
                    <span className="size-1.5 rounded-full bg-accent motion-safe:animate-pulse" />
                  ) : null}
                </span>
                {name}
              </li>
            ))}
          </ol>
        </>
      )}

      <p className="mt-5 text-meta text-mute">
        {phase.at === "refused"
          ? phase.message
          : phase.at === "lost"
            ? LOST
            : phase.at === "reading"
              ? "You can leave this page. Illarin keeps reading the file."
              : phase.at === "opening"
                ? "Your draft is private until you publish it."
                : "Keep this page open until the upload finishes."}
      </p>

      {refused ? (
        <Button className="mt-5" onClick={onBeginAgain} variant="primary">
          Choose another file
        </Button>
      ) : phase.at === "lost" ? (
        <Button className="mt-5" onClick={onCheckAgain} variant="primary">
          Check again
        </Button>
      ) : null}
    </motion.section>
  );
}

function Gate({
  action,
  children,
  heading,
  href,
}: {
  action: string;
  children: string;
  heading: string;
  href: string;
}) {
  return (
    <section className="mt-10 rounded-plate bg-deep p-6">
      <h2 className="font-display text-section font-medium text-ink">
        {heading}
      </h2>
      <p className="mt-2 text-ui text-mute">{children}</p>
      <Button asChild className="mt-5" variant="primary">
        <Link href={href}>{action}</Link>
      </Button>
    </section>
  );
}
