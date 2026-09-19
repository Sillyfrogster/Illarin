"use client";

import { useId, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextArea, TextInput, Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { Select } from "@/components/ui/select";
import {
  type DiscordRepair,
  type DiscordRepairResult,
  repairDiscordAnnouncement,
} from "@/lib/api/publication";
import type { BlogAnnouncementAttempt } from "@/lib/api/query";

export function DiscordRepairControls({
  attempt,
}: {
  attempt: BlogAnnouncementAttempt;
}) {
  const prefix = useId();
  const [action, setAction] = useState<DiscordRepair["action"]>("edit");
  const [messageId, setMessageId] = useState(attempt.messageId ?? "");
  const [text, setText] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const [request, setRequest] = useState<DiscordRepair | null>(null);
  const [result, setResult] = useState<DiscordRepairResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    if (busy || result) return;
    const repair = request ?? {
      action,
      messageId,
      text,
      requestId: crypto.randomUUID(),
    };
    setRequest(repair);
    setBusy(true);
    try {
      const answer = await repairDiscordAnnouncement(attempt.id, repair);
      setError(answer.error ?? "");
      if (answer.value) setResult(answer.value);
    } catch {
      setError(
        "Illarin could not confirm the repair. Check this request again before starting a new action.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-3">
      <MorphingDisclosure summary="Repair Discord announcement">
        <form
          className="mt-4 grid max-w-xl gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            setReviewing(true);
          }}
        >
          <p className="font-prose text-meta text-mute">
            Check {attempt.integration} first. This changes Discord only. Copies
            readers have already received remain outside Illarin’s control.
          </p>
          <fieldset
            disabled={reviewing || busy}
            className="grid gap-4 disabled:opacity-70"
          >
            <Field htmlFor={`${prefix}-action`} label="Action">
              <Select
                id={`${prefix}-action`}
                value={action}
                onChange={(event) =>
                  setAction(event.target.value as DiscordRepair["action"])
                }
              >
                <option value="edit">Edit the announcement note</option>
                <option value="delete">Delete the original message</option>
                <option value="correction">Send a correction</option>
              </Select>
            </Field>
            {action !== "correction" ? (
              <Field
                htmlFor={`${prefix}-message`}
                label="Discord message ID"
                hint="In Discord, enable Developer Mode, then right-click the message and choose Copy Message ID."
              >
                <TextInput
                  id={`${prefix}-message`}
                  value={messageId}
                  readOnly={Boolean(attempt.messageId)}
                  required
                  pattern="[0-9]{17,20}"
                  onChange={(event) => setMessageId(event.target.value)}
                />
              </Field>
            ) : null}
            {action !== "delete" ? (
              <Field
                htmlFor={`${prefix}-text`}
                label={action === "edit" ? "Replacement note" : "Correction"}
                hint="Mentions are disabled. The original post details stay attached."
              >
                <TextArea
                  id={`${prefix}-text`}
                  value={text}
                  required
                  maxLength={1800}
                  onChange={(event) => setText(event.target.value)}
                />
              </Field>
            ) : null}
          </fieldset>
          {error ? <Trouble>{error}</Trouble> : null}
          {result ? (
            <output className="font-prose text-ui" aria-live="polite">
              {result.detail}
              {result.messageId && action === "correction"
                ? ` Message ID: ${result.messageId}.`
                : ""}
            </output>
          ) : null}
          {reviewing && !result ? (
            <p className="font-prose text-ui">
              {action === "correction"
                ? "This sends a new message. If an earlier request reached Discord without confirmation, another message may create a duplicate."
                : action === "delete"
                  ? `Delete message ${messageId} from ${attempt.integration}? This cannot be undone.`
                  : `Replace the note on message ${messageId} in ${attempt.integration}?`}
            </p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            {!reviewing ? (
              <Button type="submit">Review action</Button>
            ) : !result ? (
              <Button
                loading={busy}
                variant={action === "delete" ? "stop" : "primary"}
                onClick={() => void submit()}
              >
                {request
                  ? "Check this request"
                  : action === "delete"
                    ? "Delete message"
                    : action === "edit"
                      ? "Save Discord note"
                      : "Send correction"}
              </Button>
            ) : null}
            {reviewing && !request ? (
              <Button onClick={() => setReviewing(false)}>Back</Button>
            ) : null}
            {result || error ? (
              <Button
                onClick={() => {
                  setReviewing(false);
                  setRequest(null);
                  setResult(null);
                  setError("");
                }}
              >
                New action
              </Button>
            ) : null}
          </div>
        </form>
      </MorphingDisclosure>
    </div>
  );
}
