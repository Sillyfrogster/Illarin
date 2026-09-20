"use client";

import { Shield } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import { takeDownWork } from "@/lib/api/query";
import { Field, TextAreaField } from "./workspace/fields";

export function TakedownControl({
  workId,
  creator,
  typeName,
}: {
  workId: string;
  creator: string;
  typeName: string;
}) {
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  function ask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!reason.trim() || pending) return;
    setMessage("");
    setConfirming(true);
  }

  async function takeDown() {
    setPending(true);
    setMessage("");
    try {
      await takeDownWork(workId, reason.trim());
      router.replace("/browse");
    } catch {
      setMessage(
        `The ${typeName} could not be taken down. Your reason is still here.`,
      );
      setConfirming(false);
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="takedown-heading" className="flex gap-3">
      <span aria-hidden="true" className="mt-0.5 shrink-0 text-mute">
        <Shield size={18} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div>
          <h3 className="text-ui font-medium text-ink" id="takedown-heading">
            Take down this {typeName}
          </h3>
          <p className="mt-1 text-meta text-mute">
            It leaves Browse and answers as missing to everyone but @{creator},
            who keeps reading and downloading their own file but cannot edit it,
            relist it or delete it.
          </p>
        </div>
        <form className="flex flex-col gap-3" onSubmit={ask}>
          <Field label="Reason shown to the creator">
            <TextAreaField
              disabled={pending}
              onChange={(event) => setReason(event.target.value)}
              required
              rows={3}
              value={reason}
            />
          </Field>
          {message ? (
            <p className="text-meta text-stop" role="alert">
              {message}
            </p>
          ) : null}
          {confirming ? (
            <div className="flex flex-col gap-3 rounded-plate bg-stop-wash p-4">
              <p className="text-ui text-ink">
                Take down this {typeName}? Only its creator will be able to open
                its page.
              </p>
              <div className="flex flex-wrap items-center gap-2">
                <Button loading={pending} onClick={takeDown} variant="stop">
                  Take down {typeName}
                </Button>
                <Button
                  disabled={pending}
                  onClick={() => setConfirming(false)}
                  variant="ghost"
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <Button
              className="self-start"
              disabled={!reason.trim()}
              type="submit"
            >
              Take down {typeName}
            </Button>
          )}
        </form>
      </div>
    </section>
  );
}
