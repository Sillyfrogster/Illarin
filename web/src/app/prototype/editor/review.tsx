"use client";

import { ArrowRight, Check, FileUp } from "lucide-react";
import { AnimatedNumber } from "./animated-number";
import type { Asset, Change, Notes } from "./data";
import {
  Button,
  Eyebrow,
  Field,
  Input,
  Notice,
  Select,
  SpectralButton,
  Textarea,
} from "./ui";

function verb(change: Change) {
  if (!change.before) return "Added";
  if (!change.after) return "Removed";
  return "Changed";
}

export function ChangeList({ changes }: { changes: Change[] }) {
  if (!changes.length)
    return (
      <Notice>
        No changes yet. Your private work and the published version match.
      </Notice>
    );
  return (
    <div className="ws:space-y-7">
      {[...new Set(changes.map((change) => change.group))].map((group) => (
        <section key={group}>
          <Eyebrow className="ws:mb-2">{group}</Eyebrow>
          {changes
            .filter((change) => change.group === group)
            .map((change) => (
              <details
                key={change.id}
                className="ws:group ws:shadow-[inset_0_-1px_0_var(--w-hairline)]"
              >
                <summary className="ws:flex ws:min-h-12 ws:cursor-pointer ws:list-none ws:items-center ws:justify-between ws:gap-3 ws:py-2.5 ws:text-sm ws:font-semibold">
                  <span className="ws:min-w-0 ws:wrap-anywhere">
                    {change.label}
                  </span>
                  <span className="ws:shrink-0 ws:rounded-full ws:bg-ink/8 ws:px-2.5 ws:py-1 ws:text-[0.6875rem] ws:font-bold ws:tracking-wide ws:text-mute ws:uppercase">
                    {verb(change)}
                  </span>
                </summary>
                <div className="ws:grid ws:gap-3 ws:pb-4 ws:md:grid-cols-2">
                  <div className="ws:min-w-0 ws:rounded-xl ws:bg-ink/5 ws:p-4 ws:shadow-[inset_2px_0_0_var(--w-line)]">
                    <Eyebrow>Published now</Eyebrow>
                    <p className="ws:mt-2 ws:max-h-72 ws:overflow-auto ws:text-sm ws:leading-7 ws:whitespace-pre-wrap ws:wrap-anywhere ws:text-mute">
                      {change.before || "Not present"}
                    </p>
                  </div>
                  <div
                    className="ws:min-w-0 ws:rounded-xl ws:bg-card ws:p-4 ws:shadow-[inset_0_0_0_1px_var(--w-line)]"
                    style={{
                      borderInlineStartWidth: 2,
                      borderImage: "var(--w-spectrum) 1",
                    }}
                  >
                    <Eyebrow>After this update</Eyebrow>
                    <p className="ws:mt-2 ws:max-h-72 ws:overflow-auto ws:text-sm ws:leading-7 ws:whitespace-pre-wrap ws:wrap-anywhere">
                      {change.after || "Removed"}
                    </p>
                  </div>
                </div>
              </details>
            ))}
        </section>
      ))}
    </div>
  );
}

export function UpdateReview({
  asset,
  notes,
  setNotes,
  setVersion,
  changes,
  reviewed,
  issues,
  readOnly,
  listed,
  keepEditing,
  check,
  publish,
  busy,
}: {
  asset: Asset;
  notes: Notes;
  setNotes: (notes: Notes) => void;
  setVersion: (version: string) => void;
  changes: Change[];
  reviewed: boolean;
  issues: string[];
  readOnly: boolean;
  listed: boolean;
  keepEditing: () => void;
  check: () => void;
  publish: () => void;
  busy: boolean;
}) {
  return (
    <div className="ws:space-y-7">
      <div className="ws:flex ws:items-center ws:gap-3">
        <span className="ws:font-display ws:text-4xl ws:leading-none">
          <AnimatedNumber value={changes.length} />
        </span>
        <span className="ws:text-sm ws:text-mute">
          computed {changes.length === 1 ? "change" : "changes"} since the
          published version
        </span>
      </div>

      {issues.length > 0 && (
        <Notice tone="critical">
          <p className="ws:font-bold">This update needs attention</p>
          <ul className="ws:mt-2 ws:list-disc ws:space-y-1 ws:pl-5">
            {issues.map((issue) => (
              <li key={issue}>{issue}</li>
            ))}
          </ul>
          <Button
            size="small"
            variant="outline"
            className="ws:mt-3"
            onClick={keepEditing}
          >
            Back to the page
          </Button>
        </Notice>
      )}

      {reviewed && (
        <Notice tone="amber">
          <div className="ws:flex ws:items-center ws:gap-2 ws:font-bold">
            <Check className="ws:size-4" />
            Ready for your final decision
          </div>
          <p className="ws:mt-1">
            The content and notes here are the exact candidate that publication
            will make public.
          </p>
          <div className="ws:mt-3 ws:flex ws:flex-wrap ws:gap-2">
            <SpectralButton onClick={publish} disabled={busy || readOnly}>
              Publish update <ArrowRight />
            </SpectralButton>
            <Button size="small" variant="outline" onClick={keepEditing}>
              Keep editing
            </Button>
          </div>
        </Notice>
      )}

      {!reviewed && (
        <div className="ws:flex ws:flex-wrap ws:items-center ws:gap-3">
          <SpectralButton onClick={check} disabled={busy || readOnly}>
            Check this update
            <ArrowRight />
          </SpectralButton>
          <span className="ws:text-sm ws:text-mute">
            Nothing is published until you decide after the check.
          </span>
        </div>
      )}

      <div className="ws:grid ws:gap-10 ws:lg:grid-cols-2">
        <fieldset
          disabled={readOnly || reviewed}
          className="ws:min-w-0 ws:space-y-6"
        >
          <Field
            label="Update summary"
            hint="Required. One short explanation for readers."
          >
            <Input
              value={notes.summary}
              onChange={(e) => setNotes({ ...notes, summary: e.target.value })}
              placeholder="A new way to arrive, and a little more mystery"
            />
          </Field>
          <Field
            label="Update notes"
            hint="Optional. Your words; computed changes appear alongside them."
          >
            <Textarea
              rows={5}
              value={notes.notes}
              onChange={(e) => setNotes({ ...notes, notes: e.target.value })}
              placeholder="What should a returning reader know?"
            />
          </Field>
          <Field
            label="Version label"
            hint="Optional free text. Repeated labels are allowed."
          >
            <Input
              value={asset.version}
              onChange={(e) => setVersion(e.target.value)}
            />
          </Field>
          <Notice>
            Quiet publication. This prototype sends no announcements.
            {!listed &&
              " The asset is unlisted and stays reachable by direct link."}
          </Notice>
        </fieldset>
        <div className="ws:min-w-0">
          <ChangeList changes={changes} />
        </div>
      </div>
    </div>
  );
}

export function ReplacementReview({
  changes,
  choice,
  setChoice,
  accept,
  cancel,
  busy,
  kind,
}: {
  changes: Change[];
  choice: string;
  setChoice: (value: string) => void;
  accept: () => void;
  cancel: () => void;
  busy: boolean;
  kind: Asset["kind"];
}) {
  return (
    <div className="ws:space-y-6">
      <p className="ws:text-sm ws:text-mute">
        {kind === "character"
          ? "morrow-next.json · Character Card v2"
          : "atlas-next.json · Lorebook JSON"}
        . This sample is already parsed. No file leaves your browser.
      </p>
      <Notice tone="amber">
        <div className="ws:flex ws:gap-3">
          <FileUp className="ws:size-5 ws:shrink-0" />
          <div>
            <p className="ws:font-semibold">
              Incoming content replaces the matching working-copy content.
            </p>
            <p>
              Your writing in those fields will be replaced, including
              conflicting edits. Illarin-only notes and page arrangement are
              kept.
            </p>
          </div>
        </div>
      </Notice>
      {kind === "character" && (
        <Field
          label="This format cannot carry group-only greetings"
          hint="Choose explicitly. Keeping them preserves them in Illarin; they still cannot travel in a CCv2 export."
        >
          <Select value={choice} onChange={(e) => setChoice(e.target.value)}>
            <option value="">Choose what to do</option>
            <option value="keep">Keep group-only greetings on Illarin</option>
            <option value="remove">
              Remove group-only greetings from the working copy
            </option>
          </Select>
        </Field>
      )}
      <ChangeList changes={changes} />
      <div className="ws:flex ws:flex-wrap ws:gap-3">
        <Button variant="primary" onClick={accept} disabled={busy}>
          {busy ? "Accepting…" : "Accept into private work"}
          <ArrowRight />
        </Button>
        <Button variant="outline" onClick={cancel}>
          Cancel replacement
        </Button>
      </div>
      <p className="ws:text-sm ws:text-mute">
        Acceptance saves a private candidate. Publication still needs an update
        summary and a final review. The existing public asset stays available.
      </p>
    </div>
  );
}
