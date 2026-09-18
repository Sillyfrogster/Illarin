"use client";

import type { LucideIcon } from "lucide-react";
import { ChevronRight, FileUp, Send } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import {
  fetchWaitingReplacement,
  fetchWork,
  type IngestOperation,
  publishWork,
  type ReadinessItem,
  type VersionChangeGroup,
} from "@/lib/api/query";
import type { PageTarget, ReadinessTarget } from "@/lib/readiness";
import { reviewBlockedReason, updateStanding } from "@/lib/work-publication";
import { useWorkingCopy, WORKING_COPY_SAVED } from "@/lib/working-copy";
import { Note } from "./fields";
import { ReadinessList } from "./ReadinessList";
import { ReplacementStep } from "./ReplacementStep";
import { ReviewStep } from "./ReviewStep";
import { useWorkspace } from "./state";

type Step = "home" | "replace" | "review" | "confirm";

export function PublicationRail({
  typeName,
  onGo,
  readiness,
  unlisted,
  unpublishedChanges,
}: {
  typeName: string;
  onGo: (target: PageTarget) => void;
  readiness: ReadinessItem[] | undefined;
  unlisted: boolean;
  unpublishedChanges: boolean;
}) {
  const workspace = useWorkspace();
  const router = useRouter();
  const [step, setStep] = useState<Step>("home");
  const [changed, setChanged] = useState(unpublishedChanges);
  const [waiting, setWaiting] = useState<IngestOperation | null>(null);
  const [applied, setApplied] = useState<VersionChangeGroup[] | null>(null);

  useEffect(() => setChanged(unpublishedChanges), [unpublishedChanges]);

  const readStanding = useCallback(async () => {
    const page = await fetchWork(workspace.workId, undefined, true);
    if (page) setChanged(Boolean(page.unpublishedChanges));
  }, [workspace.workId]);

  useEffect(() => {
    const saved = () => void readStanding();
    window.addEventListener(WORKING_COPY_SAVED, saved);
    return () => window.removeEventListener(WORKING_COPY_SAVED, saved);
  }, [readStanding]);

  useEffect(() => {
    if (workspace.isDraft) return;
    let reading = true;
    void fetchWaitingReplacement(workspace.workId).then((found) => {
      if (reading) setWaiting(found);
    });
    return () => {
      reading = false;
    };
  }, [workspace.workId, workspace.isDraft]);

  function settled() {
    setWaiting(null);
    setStep("home");
    void readStanding();
    router.refresh();
  }

  function goTo(target: ReadinessTarget) {
    if (target.where === "replacement") {
      setStep("replace");
      return;
    }
    onGo(target);
  }

  if (workspace.isDraft) {
    return (
      <DraftPublication
        confirming={step === "confirm"}
        typeName={typeName}
        onConfirm={() => setStep("confirm")}
        onGo={goTo}
        onLeaveConfirm={() => setStep("home")}
        readiness={readiness ?? []}
      />
    );
  }

  if (step === "replace") {
    return (
      <ReplacementStep
        onApplied={(changes) => {
          setApplied(changes);
          settled();
        }}
        onBack={() => setStep("home")}
        onDiscarded={settled}
        onWaiting={setWaiting}
        waiting={waiting}
      />
    );
  }

  if (step === "review") {
    return (
      <ReviewStep
        applied={applied}
        typeName={typeName}
        onBack={() => setStep("home")}
        onGo={goTo}
        onPublished={() => {
          setApplied(null);
          settled();
        }}
        unlisted={unlisted}
      />
    );
  }

  const blocked = reviewBlockedReason(waiting, changed);
  const shortfall = (readiness ?? []).filter((item) => !item.met);

  return (
    <div className="flex flex-col gap-7">
      <p className="text-ui text-ink">{updateStanding(waiting, changed)}</p>

      <div className="flex flex-col gap-2">
        <Path
          detail={
            blocked || "Write the summary readers see, then hand them the page."
          }
          disabled={blocked !== ""}
          icon={Send}
          name="Review and publish an update"
          onClick={() => setStep("review")}
        />
        <Path
          detail={
            waiting
              ? "An upload is waiting on you."
              : "Everything a file carries replaces what is on this page."
          }
          icon={FileUp}
          name={waiting ? "Review the uploaded file" : "Replace the file"}
          onClick={() => setStep("replace")}
        />
      </div>

      {shortfall.length > 0 ? (
        <section className="flex flex-col gap-4">
          <h3 className="font-display text-ui font-medium text-ink">
            Missing recommended content
          </h3>
          <Note>
            These fields are required when publishing a new {typeName}. Your
            existing page stays public.
          </Note>
          <ReadinessList items={shortfall} onGo={goTo} />
        </section>
      ) : null}
    </div>
  );
}

function DraftPublication({
  confirming,
  typeName,
  onConfirm,
  onGo,
  onLeaveConfirm,
  readiness,
}: {
  confirming: boolean;
  typeName: string;
  onConfirm: () => void;
  onGo: (target: ReadinessTarget) => void;
  onLeaveConfirm: () => void;
  readiness: ReadinessItem[];
}) {
  const workspace = useWorkspace();
  const candidate = useWorkingCopy();
  const router = useRouter();
  const [items, setItems] = useState(readiness);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => setItems(readiness), [readiness]);

  const missing = items.filter((item) => !item.met);

  async function publish() {
    if (busy) return;
    setBusy(true);
    setMessage("");
    const answer = await publishWork(candidate, workspace.workId);
    setBusy(false);
    if (answer.published) {
      workspace.closePane();
      router.refresh();
      return;
    }
    if (answer.readiness) {
      setItems(answer.readiness);
      onLeaveConfirm();
      return;
    }
    setMessage(answer.error);
  }

  if (confirming) {
    return (
      <div className="flex flex-col gap-5">
        <RailBack onClick={onLeaveConfirm}>Publication</RailBack>
        <h3 className="font-display text-section font-medium text-ink">
          Publish this {typeName}?
        </h3>
        <p className="text-ui text-ink">
          This becomes a public page anyone can open. It is one-way, and a
          published {typeName} never returns to a draft.
        </p>
        {message ? (
          <p
            className="rounded-control bg-stop-wash p-3 text-meta text-ink"
            role="alert"
          >
            {message}
          </p>
        ) : null}
        <div className="flex flex-wrap items-center gap-2">
          <Button loading={busy} onClick={publish} variant="primary">
            {busy ? "Publishing…" : "Publish"}
          </Button>
          <Button disabled={busy} onClick={onLeaveConfirm} variant="ghost">
            Back to the draft
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-7">
      <p className="text-ui text-ink">
        Only you can open this page. It is in no browse or search result, and
        nobody else can download it.
      </p>

      <section className="flex flex-col gap-4">
        <h3 className="font-display text-ui font-medium text-ink">
          Before you publish
        </h3>
        <ReadinessList items={items} onGo={onGo} />
      </section>

      <div className="flex flex-col gap-3">
        <Button
          className="self-start"
          disabled={missing.length > 0}
          onClick={onConfirm}
          variant="primary"
        >
          Publish
        </Button>
        <Note>
          {missing.length === 0
            ? `A published ${typeName} cannot return to draft. A blurb is optional.`
            : missing.length === 1
              ? "One thing above is still missing."
              : `${missing.length} things above are still missing.`}
        </Note>
      </div>
    </div>
  );
}

function Path({
  detail,
  disabled,
  icon: Icon,
  name,
  onClick,
}: {
  detail: string;
  disabled?: boolean;
  icon: LucideIcon;
  name: string;
  onClick: () => void;
}) {
  return (
    <button
      className="flex w-full items-start gap-3 rounded-plate bg-deep p-4 text-left outline-offset-3 hover:bg-rule/45 disabled:opacity-45 disabled:hover:bg-deep"
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      <Icon
        aria-hidden="true"
        className="mt-0.5 shrink-0 text-accent"
        size={18}
      />
      <span className="min-w-0 flex-1">
        <span className="block text-ui font-medium text-ink">{name}</span>
        <span className="mt-1 block text-meta text-mute">{detail}</span>
      </span>
      <ChevronRight
        aria-hidden="true"
        className="mt-0.5 shrink-0 text-mute"
        size={16}
      />
    </button>
  );
}
