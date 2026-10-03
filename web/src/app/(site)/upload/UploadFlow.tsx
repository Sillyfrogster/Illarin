"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { Upload as UploadIcon } from "lucide-react";
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
import { timing } from "@/lib/timing";
import { workHref } from "@/lib/work-url";
import { DropStage } from "./DropStage";
import { ScanView, type Upload } from "./ScanView";
import { StartFromNothing } from "./StartFromNothing";
import { revokeScan, type ScanPart, scanFile } from "./scan-file";
import { sendFile } from "./send-file";

const POLL_MS = 600;

/** How long the found work stays on screen before its draft opens. */
const FOUND_MS = 1100;

type Phase =
  | { at: "choosing" }
  | {
      at: "busy";
      file: File;
      upload: Upload;
      operation?: UploadOperation;
      found?: string;
    };

const SWAP = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -8 },
  transition: timing.settle,
} as const;

/** UploadFlow offers the two starts side by side, sends a chosen file straight away, scans it on screen while it travels, and opens the draft it becomes. */
export function UploadFlow({ choices }: { choices: BuildChoices | null }) {
  const { account } = useAuth();
  const router = useRouter();
  const still = useReducedMotion();
  const [phase, setPhase] = useState<Phase>({ at: "choosing" });
  const [parts, setParts] = useState<ScanPart[] | null>(null);
  const [scanned, setScanned] = useState(false);
  const top = useRef<HTMLDivElement>(null);
  const over = useFileOverWindow(
    phase.at === "choosing" && Boolean(account?.emailVerified),
    send,
  );

  useEffect(() => () => revokeScan(parts ?? []), [parts]);

  const busy = phase.at === "busy" ? phase : null;
  const polling = busy?.upload.at === "reading" && busy.operation ? busy : null;
  useEffect(() => {
    if (!polling?.operation) return;
    const operation = polling.operation;
    let active = true;
    const timer = setTimeout(async () => {
      try {
        const next = await readUploadOperation(operation.url);
        if (!active) return;
        if (next.status === "failed") {
          setPhase({
            ...polling,
            upload: {
              at: "refused",
              message:
                next.failure?.message ?? "Illarin could not read this file.",
            },
          });
        } else if (next.status === "success" && next.work) {
          setPhase({
            ...polling,
            upload: {
              at: "found",
              name: next.work.name,
              type: next.work.type as BrowseType,
            },
            found: workHref(next.work.id, next.work.name),
          });
        } else {
          setPhase({ ...polling, operation: next });
        }
      } catch {
        if (active) setPhase({ ...polling, upload: { at: "lost" } });
      }
    }, POLL_MS);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [polling]);

  const opening = busy?.found && scanned ? busy.found : null;
  useEffect(() => {
    if (!opening) return;
    const timer = setTimeout(() => router.push(opening), FOUND_MS);
    return () => clearTimeout(timer);
  }, [opening, router]);

  async function send(file: File) {
    setParts(null);
    setScanned(false);
    setPhase({ at: "busy", file, upload: { at: "sending", sent: 0 } });
    top.current?.scrollIntoView({
      behavior: still ? "auto" : "smooth",
      block: "center",
    });
    void scanFile(file).then(setParts);
    const sent = await sendFile(file, (fraction) =>
      setPhase((current) =>
        current.at === "busy" && current.upload.at === "sending"
          ? { ...current, upload: { at: "sending", sent: fraction } }
          : current,
      ),
    );
    setPhase(
      sent.operation
        ? {
            at: "busy",
            file,
            upload: { at: "reading" },
            operation: sent.operation,
          }
        : { at: "busy", file, upload: { at: "refused", message: sent.error } },
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
        href="/sign-in?returnTo=%2Fupload"
        heading="Sign in to publish"
      />
    );
  }

  if (!account.emailVerified) {
    return (
      <Gate
        action="Verify email"
        href="/verify-email?returnTo=%2Fupload"
        heading="Verify your email to publish"
      >
        Open the link Illarin emailed you.
      </Gate>
    );
  }

  return (
    <div className="mt-8 scroll-mt-32" ref={top}>
      <AnimatePresence initial={false} mode="wait">
        <motion.div
          key={phase.at === "choosing" ? "stage" : "file"}
          {...(still ? {} : SWAP)}
        >
          {phase.at === "choosing" ? (
            <div className="grid items-stretch gap-4 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
              <DropStage onFile={send} over={over} />
              <StartFromNothing choices={choices} />
            </div>
          ) : (
            <div className="rounded-card bg-inset p-6 sm:p-8">
              <ScanView
                file={phase.file}
                onBeginAgain={() => setPhase({ at: "choosing" })}
                onCheckAgain={() =>
                  setPhase({ ...phase, upload: { at: "reading" } })
                }
                onScanned={() => setScanned(true)}
                parts={parts}
                upload={phase.upload}
              />
            </div>
          )}
        </motion.div>
      </AnimatePresence>

      <AnimatePresence>
        {over ? (
          <motion.div
            animate={{ opacity: 1 }}
            aria-hidden="true"
            className="pointer-events-none fixed inset-0 z-85 bg-field/90 p-4 sm:p-6"
            exit={{ opacity: 0 }}
            initial={{ opacity: 0 }}
            transition={timing.quick}
          >
            <div className="flex size-full flex-col items-center justify-center gap-4 rounded-card border border-dashed border-accent bg-accent-wash/60">
              <span className="flex size-12 items-center justify-center rounded-plate bg-action text-on-accent">
                <UploadIcon className="size-5" strokeWidth={1.8} />
              </span>
              <p className="font-display text-title font-medium text-ink">
                Let go to upload
              </p>
            </div>
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
  children?: string;
  heading: string;
  href: string;
}) {
  return (
    <section className="mt-10 rounded-card bg-inset p-6">
      <h2 className="font-display text-section font-medium text-ink">
        {heading}
      </h2>
      {children ? <p className="mt-2 text-ui text-mute">{children}</p> : null}
      <Button asChild className="mt-5" variant="primary">
        <Link href={href}>{action}</Link>
      </Button>
    </section>
  );
}
