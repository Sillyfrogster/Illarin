"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { type IngestOperation, readIngestOperation } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { importStage } from "@/lib/import-stage";
import { ImportFile } from "./ImportFile";
import { ImportReceipt } from "./ImportReceipt";
import { ImportRefusal, ImportWatch } from "./ImportWatch";
import { StartFromNothing } from "./StartFromNothing";

const POLL_MS = 600;

const LOST =
  "The connection was interrupted. Your file is safe; check again to see where it got to.";

export function UploadFlow() {
  const { account } = useAuth();
  const heading = useRef<HTMLHeadingElement>(null);
  const [operation, setOperation] = useState<IngestOperation | null>(null);
  const [message, setMessage] = useState("");

  const stage = importStage(operation, message);
  const watching = stage.at === "reading";

  useEffect(() => {
    if (!operation || !watching) return;
    const current = operation;
    let active = true;
    async function poll() {
      while (active) {
        await new Promise((resolve) => setTimeout(resolve, POLL_MS));
        if (!active) return;
        try {
          const next = await readIngestOperation(current.url);
          if (!active) return;
          setOperation(next);
          if (next.status !== "pending" && next.status !== "processing") return;
        } catch {
          if (active) setMessage(LOST);
          return;
        }
      }
    }
    void poll();
    return () => {
      active = false;
    };
  }, [operation, watching]);

  useEffect(() => {
    if (stage.at === "refused" || stage.at === "arrived")
      heading.current?.focus();
  }, [stage.at]);

  const beginAgain = useCallback(() => {
    setOperation(null);
    setMessage("");
  }, []);

  if (account === undefined) {
    return (
      <output className="mt-8 block text-ui text-mute">
        Reading your account…
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
        Your account keeps every creation tied to the creator who made it.
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
        Verification puts every public file behind an address you control.
      </Gate>
    );
  }

  if (stage.at === "refused") {
    return (
      <ImportRefusal
        headingRef={heading}
        message={stage.message}
        onBeginAgain={beginAgain}
      />
    );
  }

  if (stage.at === "arrived") {
    return (
      <ImportReceipt
        asset={stage.asset}
        headingRef={heading}
        onBeginAgain={beginAgain}
      />
    );
  }

  if (stage.at === "reading" || stage.at === "lost") {
    return (
      <ImportWatch
        onCheckAgain={() => {
          setMessage("");
          setOperation((current) => (current ? { ...current } : current));
        }}
        stage={stage}
      />
    );
  }

  return (
    <div className="mt-10 flex flex-col gap-12">
      <ImportFile onAccepted={setOperation} />
      <StartFromNothing />
    </div>
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
