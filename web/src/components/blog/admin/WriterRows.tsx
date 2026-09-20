"use client";

import { UserRoundPlus } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import {
  Nothing,
  PanelHead,
  Row,
  Rows,
  StartAction,
} from "@/components/register/RowParts";
import { Consequence, StepForm } from "@/components/register/StepParts";
import { Field, TextInput } from "@/components/ui/field";
import { switchWriterOff, switchWriterOn } from "@/lib/api/blog";
import type { WriterResponse } from "@/lib/api/query";
import { nothingIn } from "@/lib/blog-admin";
import { readableDate } from "@/lib/dates";

export function WriterRows({
  writers,
  onOpen,
}: {
  writers: WriterResponse[];
  onOpen: (writer: WriterResponse | null) => void;
}) {
  return (
    <>
      <PanelHead
        action={
          <StartAction icon={UserRoundPlus} onClick={() => onOpen(null)}>
            Switch on a writer
          </StartAction>
        }
        id="register-heading"
        title="Writers"
      />

      {writers.length === 0 ? (
        <Nothing>{nothingIn("writers")}</Nothing>
      ) : (
        <Rows>
          {writers.map((writer) => (
            <Row
              facts={
                <>
                  <span>@{writer.handle}</span>
                  <span>Writing since {readableDate(writer.since)}</span>
                </>
              }
              key={writer.accountId}
              lead={
                <CreatorPortrait
                  handle={writer.handle}
                  picture={writer.avatar}
                  size="sm"
                />
              }
              onOpen={() => onOpen(writer)}
              open={`Switch off @${writer.handle}`}
              title={writer.displayName || `@${writer.handle}`}
            />
          ))}
        </Rows>
      )}
    </>
  );
}

export function WriterStep({
  existing,
  onClose,
  onFailure,
  onSwitchedOff,
  onSwitchedOn,
}: {
  existing: WriterResponse | null;
  onClose: () => void;
  onFailure: (message: string) => void;
  onSwitchedOff: () => void;
  onSwitchedOn: (writer: WriterResponse) => void;
}) {
  const [handle, setHandle] = useState("");
  const [busy, setBusy] = useState(false);

  async function switchOn() {
    setBusy(true);
    const answer = await switchWriterOn(handle.trim().replace(/^@/, ""));
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onSwitchedOn(answer.value);
    onClose();
  }

  async function switchOff() {
    if (!existing) return;
    setBusy(true);
    const answer = await switchWriterOff(existing.accountId);
    setBusy(false);
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    onSwitchedOff();
    onClose();
  }

  if (existing) {
    return (
      <>
        <Link
          className="mb-5 inline-flex min-h-11 items-center font-ui text-meta font-medium text-accent underline-offset-4 outline-offset-3 hover:underline"
          href={`/@${existing.handle}`}
        >
          View profile
        </Link>
        <Consequence
          action="Switch off"
          busy={busy}
          confirm="Switch off writing"
          onConfirm={switchOff}
        >
          @{existing.handle} loses the editor and any scheduled publication
          stops. Everything they published stays, under their name.
        </Consequence>
      </>
    );
  }

  return (
    <StepForm
      busy={busy}
      commit="Switch on"
      onCommit={switchOn}
      ready={Boolean(handle.trim())}
    >
      <Field
        hint="The account gains the blog editor and publishes under its own name."
        htmlFor="writer-handle"
        label="Handle"
      >
        <TextInput
          autoComplete="off"
          id="writer-handle"
          onChange={(event) => setHandle(event.target.value)}
          placeholder="kestrel.writes"
          value={handle}
        />
      </Field>
    </StepForm>
  );
}
