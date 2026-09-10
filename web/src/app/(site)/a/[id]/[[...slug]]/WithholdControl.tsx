"use client";

import { Shield } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import { withholdAsset } from "@/lib/api/query";
import { Field, TextAreaField } from "./workspace/fields";

/** The staff action that takes public access away without touching the creator's file. */
export function WithholdControl({ assetId }: { assetId: string }) {
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const decision = reason.trim();
    if (!decision || pending) return;
    setPending(true);
    setMessage("");
    try {
      await withholdAsset(assetId, decision);
      router.replace("/browse");
    } catch {
      setMessage("The asset could not be withheld. Try again.");
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="withhold-heading" className="flex gap-3">
      <span aria-hidden="true" className="mt-0.5 shrink-0 text-mute">
        <Shield size={18} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div>
          <h3 className="text-ui font-medium text-ink" id="withhold-heading">
            Staff action
          </h3>
          <p className="mt-1 text-meta text-mute">
            Remove public access without deleting the creator’s file.
          </p>
        </div>
        <form className="flex flex-col gap-3" onSubmit={submit}>
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
          <Button
            className="self-start"
            disabled={!reason.trim()}
            loading={pending}
            type="submit"
            variant="stop"
          >
            Withhold asset
          </Button>
        </form>
      </div>
    </section>
  );
}
