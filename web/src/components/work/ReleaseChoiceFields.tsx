"use client";

import type { ReactNode } from "react";
import { CheckboxRow } from "@/components/ui/checkbox";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { RadioGroup } from "@/components/ui/radio-group";

/** ReleaseChoice is which repository an extension follows and which file of each release it imports. */
export type ReleaseChoice = {
  repository: string;
  attachment: string;
  useAttachment: boolean;
  includePrereleases: boolean;
};

export const NO_RELEASE_CHOICE: ReleaseChoice = {
  repository: "",
  attachment: "",
  useAttachment: false,
  includePrereleases: false,
};

/** releaseChoiceBody is the choice as the GitHub routes read it. */
export function releaseChoiceBody(choice: ReleaseChoice) {
  return {
    repository: choice.repository,
    attachment: choice.useAttachment ? choice.attachment : null,
    includePrereleases: choice.includePrereleases,
  };
}

/** ReleaseChoiceFields asks for the repository, then the release file and prereleases, as two siblings the parent lays out. */
export function ReleaseChoiceFields({
  choice,
  hint,
  onChange,
}: {
  choice: ReleaseChoice;
  hint?: ReactNode;
  onChange: (next: ReleaseChoice) => void;
}) {
  return (
    <>
      <Field hint={hint} label="Repository URL">
        <Input
          onChange={(event) =>
            onChange({ ...choice, repository: event.target.value })
          }
          placeholder="https://github.com/owner/repository"
          required
          type="url"
          value={choice.repository}
        />
      </Field>
      <div className="flex flex-col gap-5">
        <fieldset>
          <legend className="mb-2 text-meta font-medium text-ink">
            File to import
          </legend>
          <RadioGroup
            onValueChange={(next) =>
              onChange({ ...choice, useAttachment: next === "attachment" })
            }
            options={[
              {
                value: "archive",
                label: "Source archive",
                hint: "GitHub's release archive",
              },
              {
                value: "attachment",
                label: "Named file",
                hint: "Same attachment on each release",
              },
            ]}
            value={choice.useAttachment ? "attachment" : "archive"}
          />
          {choice.useAttachment ? (
            <Input
              aria-label="Attachment file name"
              className="mt-3"
              onChange={(event) =>
                onChange({ ...choice, attachment: event.target.value })
              }
              placeholder="extension.zip"
              required
              value={choice.attachment}
            />
          ) : null}
        </fieldset>
        <CheckboxRow
          checked={choice.includePrereleases}
          label="Include prereleases"
          onCheckedChange={(checked) =>
            onChange({ ...choice, includePrereleases: checked })
          }
        />
      </div>
    </>
  );
}
