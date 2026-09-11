"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Said, Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import {
  type AssetUpdateDestinationChoice,
  readAssetUpdateDestinationChoices,
  saveAssetUpdateDestinationDefaults,
} from "@/lib/api/asset-destinations";

export function AssetDestinationDefaults({
  assetId,
  frozen,
}: {
  assetId: string;
  frozen: boolean;
}) {
  return (
    <div className="mt-7 border-t border-rule pt-5">
      <MorphingDisclosure summary="Update destinations">
        <DefaultsForm assetId={assetId} frozen={frozen} key={assetId} />
      </MorphingDisclosure>
    </div>
  );
}

function DefaultsForm({
  assetId,
  frozen,
}: {
  assetId: string;
  frozen: boolean;
}) {
  const [choices, setChoices] = useState<AssetUpdateDestinationChoice[] | null>(
    null,
  );
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      setError("");
      const answer = await readAssetUpdateDestinationChoices(assetId, signal);
      if (signal?.aborted) return;
      if (!answer.value) {
        setError(answer.error || "Could not read your destinations.");
        return;
      }
      setChoices(answer.value.destinations);
      setSelected(
        answer.value.destinations
          .filter((one) => one.byDefault)
          .map((one) => one.id),
      );
    },
    [assetId],
  );

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  async function save() {
    if (busy || frozen) return;
    setBusy(true);
    setError("");
    setNotice("");
    const answer = await saveAssetUpdateDestinationDefaults(assetId, selected);
    setBusy(false);
    if (answer.error) {
      setError(answer.error);
      return;
    }
    setChoices(
      (held) =>
        held?.map((one) => ({
          ...one,
          byDefault: selected.includes(one.id),
        })) ?? null,
    );
    setNotice(
      selected.length
        ? "Destination defaults saved for this asset."
        : "This asset has no default announcement destinations.",
    );
  }

  return (
    <div className="mt-4 grid gap-4">
      <p className="text-meta text-mute">
        Remember destinations for this asset. Saving this selection sends
        nothing. First publication never announces.
      </p>
      {frozen ? (
        <p className="text-meta text-mute">
          Destination defaults cannot change while this asset is withheld.
        </p>
      ) : null}
      {error ? <Trouble>{error}</Trouble> : null}
      {notice ? <Said>{notice}</Said> : null}
      {choices === null ? (
        error ? (
          <Button onClick={() => void load()} variant="secondary">
            Try again
          </Button>
        ) : (
          <output className="text-meta text-mute">
            Loading eligible destinations…
          </output>
        )
      ) : (
        <>
          {choices.length === 0 ? (
            <p className="text-ui text-mute">
              No verified destinations are available. Connect or verify one in
              your settings.
            </p>
          ) : (
            <fieldset className="grid gap-1" disabled={busy || frozen}>
              <legend className="sr-only">Default destinations</legend>
              {choices.map((choice) => (
                <label
                  className="flex min-h-11 cursor-pointer items-start gap-3 rounded-control px-2 py-3 hover:bg-deep"
                  key={choice.id}
                >
                  <input
                    checked={selected.includes(choice.id)}
                    className="mt-0.5 size-5 shrink-0 accent-action outline-offset-3"
                    onChange={(event) => {
                      setNotice("");
                      setSelected((current) =>
                        event.target.checked
                          ? [...current, choice.id]
                          : current.filter((id) => id !== choice.id),
                      );
                    }}
                    type="checkbox"
                  />
                  <span className="min-w-0">
                    <span className="block text-ui text-ink wrap-anywhere">
                      {choice.name}
                    </span>
                    <span className="text-meta text-mute">
                      {choice.kind === "discord"
                        ? "Discord"
                        : "Generic webhook"}
                    </span>
                  </span>
                </label>
              ))}
            </fieldset>
          )}
          <div className="flex flex-wrap gap-2">
            <Button
              disabled={busy || frozen}
              onClick={() => void save()}
              variant="primary"
            >
              {busy ? "Saving…" : "Save defaults"}
            </Button>
            {selected.length ? (
              <Button
                disabled={busy || frozen}
                onClick={() => {
                  setSelected([]);
                  setNotice("");
                }}
                variant="secondary"
              >
                Clear selection
              </Button>
            ) : null}
          </div>
        </>
      )}
      <Link
        className="inline-flex min-h-11 items-center text-meta text-accent underline-offset-4 hover:underline"
        href="/settings/update-destinations"
      >
        Manage your destinations
      </Link>
    </div>
  );
}
