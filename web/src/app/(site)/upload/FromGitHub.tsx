"use client";

import { SiGithub } from "@icons-pack/react-simple-icons";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { InputCopy } from "@/components/ui/input-copy";
import {
  NO_RELEASE_CHOICE,
  type ReleaseChoice,
  ReleaseChoiceFields,
  releaseChoiceBody,
} from "@/components/work/ReleaseChoiceFields";
import { api } from "@/lib/api/client";
import { workHref } from "@/lib/work-url";

type Started = { id: string; name: string };

function refusal(error: unknown, fallback: string): string {
  const detail = error as { error?: unknown } | null;
  return typeof detail?.error === "string" ? detail.error : fallback;
}

/** FromGitHub starts an extension draft from a repository's latest release once the repository holds the proof file. */
export function FromGitHub() {
  const router = useRouter();
  const [choice, setChoice] = useState<ReleaseChoice>(NO_RELEASE_CHOICE);
  const [proof, setProof] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function send<T>(path: string, body: unknown, fallback: string) {
    setBusy(true);
    setMessage("");
    try {
      const result = await api<T>("POST", path, { body });
      if (result.error) throw new Error(refusal(result.error, fallback));
      return result.data;
    } catch (error) {
      setMessage(error instanceof Error ? error.message : fallback);
      setBusy(false);
      return undefined;
    }
  }

  async function prove() {
    const answer = await send<{ proof: string }>(
      "/v1/github-starts/proof",
      { repository: choice.repository },
      "Illarin could not make a proof code. Try again.",
    );
    if (!answer) return;
    setProof(answer.proof);
    setBusy(false);
  }

  async function start() {
    const started = await send<Started>(
      "/v1/github-starts",
      releaseChoiceBody(choice),
      "Illarin could not start the extension. Try again.",
    );
    if (started) router.push(workHref(started.id, started.name));
  }

  return (
    <Dialog
      onOpenChange={(open) => {
        if (open) return;
        setProof("");
        setMessage("");
      }}
    >
      <DialogTrigger asChild>
        <Button variant="secondary">
          <SiGithub aria-hidden="true" className="size-4" />
          From GitHub
        </Button>
      </DialogTrigger>
      <DialogContent className="p-6 sm:p-8">
        <DialogTitle className="pr-10 font-display text-title font-medium text-ink">
          Start an extension from GitHub
        </DialogTitle>
        <DialogDescription className="mt-2 text-ui text-mute">
          Its latest release becomes your draft. Later releases import as new
          versions.
        </DialogDescription>
        <form
          className="mt-6 flex min-h-0 flex-col gap-5 overflow-y-auto"
          onSubmit={(event) => {
            event.preventDefault();
            void (proof ? start() : prove());
          }}
        >
          {proof ? (
            <div className="flex flex-col gap-4">
              <p className="text-ui text-ink">
                Add <code className="font-mono text-meta">.illarin-proof</code>{" "}
                to the root of{" "}
                <span className="wrap-anywhere">{choice.repository}</span> with
                this code inside:
              </p>
              <InputCopy value={proof} />
            </div>
          ) : (
            <ReleaseChoiceFields choice={choice} onChange={setChoice} />
          )}
          {message ? <Alert tone="stop">{message}</Alert> : null}
          <div className="flex flex-wrap justify-end gap-2">
            {proof ? (
              <Button
                disabled={busy}
                onClick={() => {
                  setProof("");
                  setMessage("");
                }}
                type="button"
                variant="ghost"
              >
                Back
              </Button>
            ) : null}
            <Button loading={busy} type="submit" variant="primary">
              {proof
                ? busy
                  ? "Importing the latest release…"
                  : "Verify and import"
                : "Get proof code"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
