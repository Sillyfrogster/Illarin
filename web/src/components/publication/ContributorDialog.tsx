"use client";

import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import {
  approveContributor,
  revokeGrant,
  setGrantDestinations,
  updateGrant,
} from "@/lib/api/publication";
import type {
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationGrant,
} from "@/lib/api/query";
import styles from "./AppDialog.module.css";
import { CategoryChoice } from "./CategoryChoice";
import tokenStyles from "./ContributorDialog.module.css";
import { DestinationChoice } from "./DestinationChoice";
import { TokenRows } from "./TokenRows";
import { useGrantTokens } from "./use-grant-tokens";

export function ContributorDialog({
  existing,
  apps,
  categories,
  destinations,
  onClose,
  onSaved,
  onRevoked,
  onFailure,
}: {
  existing: PublicationGrant | null;
  apps: PublicationApp[];
  categories: PublicationCategory[];
  destinations: PublicationDestination[];
  onClose: () => void;
  onSaved: (saved: PublicationGrant) => void;
  onRevoked: () => void;
  onFailure: (message: string) => void;
}) {
  const [handle, setHandle] = useState("");
  const [appId, setAppId] = useState("");
  const [allowed, setAllowed] = useState<string[]>(
    existing ? existing.categories.map((category) => category.id) : [],
  );
  const [fallback, setFallback] = useState(existing?.defaultCategory.id ?? "");
  const [follows, setFollows] = useState(
    existing ? existing.destinationsInherited : true,
  );
  const [reaching, setReaching] = useState<string[]>(
    existing ? existing.destinations.map((one) => one.id) : [],
  );
  const [sends, setSends] = useState<string[]>(
    existing
      ? existing.destinations
          .filter((one) => one.byDefault)
          .map((one) => one.id)
      : [],
  );
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const ready = existing
    ? allowed.length > 0 && Boolean(fallback)
    : Boolean(handle.trim()) &&
      Boolean(appId) &&
      allowed.length > 0 &&
      Boolean(fallback);

  async function save() {
    setBusy(true);
    const written = existing
      ? await updateGrant(existing.id, {
          categoryIds: allowed,
          defaultCategoryId: fallback,
        })
      : await approveContributor({
          handle: handle.trim().replace(/^@/, ""),
          appId,
          categoryIds: allowed,
          defaultCategoryId: fallback,
        });
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    const policed = await setGrantDestinations(written.value.id, {
      destinationIds: follows ? null : reaching,
      defaultDestinationIds: follows ? [] : sends,
    });
    setBusy(false);
    if (policed.error || !policed.value) {
      onSaved(written.value);
      onFailure(policed.error ?? "");
      return;
    }
    onSaved(policed.value);
    onClose();
  }

  async function revoke() {
    if (!existing) return;
    setBusy(true);
    const answer = await revokeGrant(existing.id);
    setBusy(false);
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    onRevoked();
    onClose();
  }

  return (
    <FormDialog
      open
      title={
        existing
          ? `What @${existing.holder.handle} may publish`
          : "Approve a contributor"
      }
      hint={
        existing
          ? `They publish for ${existing.app.name} under their own name. Changing the app needs a new approval.`
          : "One approval, one person, one app. They publish under their own name and gain no other authority."
      }
      commit={existing ? "Save" : "Approve"}
      busy={busy}
      ready={ready}
      onClose={onClose}
      onCommit={save}
      destructive={
        existing ? (
          <button
            type="button"
            className={styles.retire}
            onClick={() => (confirming ? revoke() : setConfirming(true))}
          >
            {confirming
              ? "Revoke, and take the badge back"
              : "Revoke the approval"}
          </button>
        ) : null
      }
    >
      {existing ? null : (
        <>
          <Field label="Handle" htmlFor="approve-handle">
            <input
              id="approve-handle"
              value={handle}
              placeholder="kestrel.writes"
              autoComplete="off"
              onChange={(event) => setHandle(event.target.value)}
            />
          </Field>
          <Field label="App" htmlFor="approve-app">
            <select
              id="approve-app"
              value={appId}
              onChange={(event) => setAppId(event.target.value)}
            >
              <option value="">Choose an app</option>
              {apps.map((app) => (
                <option key={app.id} value={app.id}>
                  {app.name}
                </option>
              ))}
            </select>
          </Field>
        </>
      )}

      <CategoryChoice
        name={existing?.id ?? "approve"}
        categories={categories}
        allowed={allowed}
        fallback={fallback}
        onAllowed={setAllowed}
        onFallback={setFallback}
      />

      <DestinationChoice
        allowed={reaching}
        defaults={sends}
        destinations={destinations}
        inherit={{ label: appPolicy(existing, apps, appId), on: follows }}
        legend="Where they may announce"
        name={existing?.id ?? "approve"}
        onAllowed={setReaching}
        onDefaults={setSends}
        onInherit={setFollows}
      />

      {existing ? <TheirTokens grant={existing} onFailure={onFailure} /> : null}

      {confirming && existing ? (
        <p className={styles.warning} role="alert">
          Everything @{existing.holder.handle} published stays, under their
          name. Press revoke again to confirm.
        </p>
      ) : null}
    </FormDialog>
  );
}

// appPolicy says what following the app means for this contributor, naming the
// app so the choice is not an abstraction.
function appPolicy(
  existing: PublicationGrant | null,
  apps: PublicationApp[],
  appId: string,
): string {
  const app = existing?.app ?? apps.find((one) => one.id === appId);
  return app
    ? `Send wherever ${app.name} sends`
    : "Send wherever the app sends";
}

function TheirTokens({
  grant,
  onFailure,
}: {
  grant: PublicationGrant;
  onFailure: (message: string) => void;
}) {
  const { tokens, live, spent, reread } = useGrantTokens(grant.id, onFailure);

  return (
    <section className={tokenStyles.theirs}>
      <h3>
        Their tokens
        {tokens ? (
          <span className={tokenStyles.count}>{live.length}</span>
        ) : null}
      </h3>
      {tokens === null ? (
        <p className={tokenStyles.quiet}>Reading their tokens…</p>
      ) : live.length === 0 ? (
        <p className={tokenStyles.quiet}>
          Nothing of theirs can reach the publication API. Only they can make a
          token; you can stop any of them.
        </p>
      ) : (
        <TokenRows
          tokens={live}
          revocable
          onRevoked={reread}
          onFailure={onFailure}
        />
      )}
      {spent.length > 0 ? (
        <p className={tokenStyles.gone}>
          {spent.length === 1
            ? "One more has already stopped working."
            : `${spent.length} more have already stopped working.`}
        </p>
      ) : null}
    </section>
  );
}
