"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  AdultContentChoice,
  AppChoice,
  adultNote,
} from "@/components/preferences/PreferenceChoices";
import {
  fetchApps,
  fetchPreferences,
  type NsfwPreference,
  saveAppPreference,
  saveNsfwPreference,
  workKeys,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";

const PREFERENCES_KEY = ["account", "preferences"] as const;

/** BrowsePreferences keeps the reader's app and adult content setting on their account. */
export function BrowsePreferences() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<
    { saved: string } | { failed: string }
  >();
  const [pending, setPending] = useState(false);
  const [picked, setPicked] = useState<{
    app?: string;
    adult?: NsfwPreference;
  }>({});

  const apps = useQuery({ queryKey: ["apps"], queryFn: fetchApps });
  const preferences = useQuery({
    enabled: Boolean(account),
    queryFn: fetchPreferences,
    queryKey: PREFERENCES_KEY,
  });

  if (!account) return null;
  if (apps.isError || preferences.isError) {
    return (
      <p className="font-ui text-ui text-stop" role="alert">
        Your browse settings could not load. Reload the page to try again.
      </p>
    );
  }
  if (!apps.data || !preferences.data) {
    return (
      <p aria-live="polite" className="font-ui text-ui text-mute">
        Loading your browse settings…
      </p>
    );
  }

  const current = preferences.data;
  const app = picked.app ?? current.app;
  const adult = picked.adult ?? current.nsfwPreference;

  async function save(
    pick: typeof picked,
    write: () => Promise<void>,
    what: string,
  ) {
    setPending(true);
    setStatus(undefined);
    setPicked(pick);
    try {
      await write();
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: PREFERENCES_KEY }),
        queryClient.invalidateQueries({ queryKey: workKeys.all }),
      ]);
      setStatus({ saved: `${what} saved.` });
    } catch {
      setPicked({});
      setStatus({ failed: `${what} could not be saved. Try again.` });
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="grid gap-8">
      <fieldset className="min-w-0 max-w-[34rem] border-0 p-0">
        <legend className="font-ui text-ui font-medium text-ink">
          Your app
        </legend>
        <p className="mt-1 font-ui text-meta text-mute">
          Browse shows only what works in your app.
        </p>
        <div className="mt-3.5">
          <AppChoice
            apps={apps.data}
            disabled={pending}
            name="settings-app"
            onChange={(next) =>
              void save(
                { app: next },
                () => saveAppPreference(next, true),
                "Your app",
              )
            }
            value={app}
          />
        </div>
      </fieldset>

      <fieldset className="min-w-0 border-0 p-0">
        <legend className="font-ui text-ui font-medium text-ink">
          Adult content
        </legend>
        <div className="mt-3">
          <AdultContentChoice
            disabled={pending}
            name="settings-adult"
            onChange={(next: NsfwPreference) =>
              void save(
                { adult: next },
                () => saveNsfwPreference(next),
                "Adult content",
              )
            }
            value={adult}
          />
        </div>
        <p className="mt-2.5 font-ui text-meta text-mute">
          {adultNote(adult)} Blur and Show are for 18 and over.
        </p>
      </fieldset>

      {status ? (
        <p
          className={
            "saved" in status
              ? "font-ui text-meta text-mute"
              : "font-ui text-meta text-stop"
          }
          role={"saved" in status ? "status" : "alert"}
        >
          {"saved" in status ? status.saved : status.failed}
        </p>
      ) : null}
    </div>
  );
}
