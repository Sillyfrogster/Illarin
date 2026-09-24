"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  type BrowseType,
  readUploadOperation,
  type UploadOperation,
} from "@/lib/api/query";
import type { BuildChoices } from "@/lib/api/shapes";
import { useAuth } from "@/lib/auth";
import { workHref } from "@/lib/work-url";
import { DropStage, FannedSheets } from "./DropStage";
import { type Reading, ReadingFile } from "./ReadingFile";
import { StartFromNothing } from "./StartFromNothing";
import { sendFile } from "./send-file";

const POLL_MS = 600;

/** How long the found work stays on screen before its draft opens. */
const FOUND_MS = 1100;

type Phase =
  | { at: "choosing" }
  | Exclude<Reading, { at: "reading" | "lost" }>
  | { at: "reading" | "lost"; file: File; operation: UploadOperation };

const SWAP = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -8 },
  transition: { duration: 0.3, ease: [0.16, 1, 0.3, 1] },
} as const;

/** UploadFlow sends a chosen file straight away, shows it being read, and opens the draft it becomes. */
export function UploadFlow({ choices }: { choices: BuildChoices | null }) {
  const { account } = useAuth();
  const router = useRouter();
  const still = useReducedMotion();
  const [phase, setPhase] = useState<Phase>({ at: "choosing" });
  const over = useFileOverWindow(
    phase.at === "choosing" && Boolean(account?.emailVerified),
    send,
  );

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
          const work = next.work;
          setPhase({
            at: "found",
            file: reading.file,
            name: work.name,
            type: work.type as BrowseType,
          });
          setTimeout(() => router.push(workHref(work.id, work.name)), FOUND_MS);
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
        <motion.div
          key={phase.at === "choosing" ? "stage" : "file"}
          {...(still ? {} : SWAP)}
        >
          {phase.at === "choosing" ? (
            <DropStage onFile={send} over={over} />
          ) : (
            <ReadingFile
              onBeginAgain={() => setPhase({ at: "choosing" })}
              onCheckAgain={() =>
                phase.at === "lost" && setPhase({ ...phase, at: "reading" })
              }
              reading={phase}
            />
          )}
        </motion.div>
      </AnimatePresence>

      {phase.at === "choosing" || phase.at === "refused" ? (
        <StartFromNothing choices={choices} />
      ) : null}

      <AnimatePresence>
        {over ? (
          <motion.div
            animate={{ opacity: 1 }}
            aria-hidden="true"
            className="pointer-events-none fixed inset-0 z-50 flex flex-col items-center justify-center gap-8 bg-field/80 backdrop-blur-md"
            exit={{ opacity: 0 }}
            initial={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          >
            <FannedSheets className="scale-125" open />
            <p className="font-display text-title font-medium text-ink">
              Let go to upload
            </p>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

/** useFileOverWindow reports a file held over the page and takes it when dropped anywhere. */
function useFileOverWindow(enabled: boolean, onFile: (file: File) => void) {
  const [over, setOver] = useState(false);
  const take = useRef(onFile);
  take.current = onFile;

  useEffect(() => {
    if (!enabled) return;
    let depth = 0;
    const holdsFile = (event: DragEvent) =>
      event.dataTransfer?.types.includes("Files") ?? false;
    const enter = (event: DragEvent) => {
      if (!holdsFile(event)) return;
      depth += 1;
      setOver(true);
    };
    const leave = (event: DragEvent) => {
      if (!holdsFile(event)) return;
      depth = Math.max(0, depth - 1);
      if (depth === 0) setOver(false);
    };
    const hover = (event: DragEvent) => {
      if (holdsFile(event)) event.preventDefault();
    };
    const drop = (event: DragEvent) => {
      if (!holdsFile(event)) return;
      event.preventDefault();
      depth = 0;
      setOver(false);
      const file = event.dataTransfer?.files[0];
      if (file) take.current(file);
    };
    window.addEventListener("dragenter", enter);
    window.addEventListener("dragleave", leave);
    window.addEventListener("dragover", hover);
    window.addEventListener("drop", drop);
    return () => {
      window.removeEventListener("dragenter", enter);
      window.removeEventListener("dragleave", leave);
      window.removeEventListener("dragover", hover);
      window.removeEventListener("drop", drop);
      setOver(false);
    };
  }, [enabled]);

  return over;
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
