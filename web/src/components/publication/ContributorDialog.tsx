"use client";

import { useCallback, useEffect, useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import {
  approveContributor,
  readTokens,
  revokeGrant,
  updateGrant,
} from "@/lib/api/publication";
import type {
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
  PublicationToken,
} from "@/lib/api/query";
import styles from "./AppDialog.module.css";
import { CategoryChoice } from "./CategoryChoice";
import tokenStyles from "./ContributorDialog.module.css";
import { TokenRows } from "./TokenRows";

export function ContributorDialog({
  existing,
  apps,
  categories,
  onClose,
  onSaved,
  onRevoked,
  onFailure,
}: {
  existing: PublicationGrant | null;
  apps: PublicationApp[];
  categories: PublicationCategory[];
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
    setBusy(false);
    if (written.error || !written.value) {
      onFailure(written.error ?? "");
      return;
    }
    onSaved(written.value);
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

function TheirTokens({
  grant,
  onFailure,
}: {
  grant: PublicationGrant;
  onFailure: (message: string) => void;
}) {
  const [tokens, setTokens] = useState<PublicationToken[] | null>(null);

  const load = useCallback(async () => {
    const answer = await readTokens(grant.id);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    setTokens(answer.value.tokens);
  }, [grant.id, onFailure]);

  useEffect(() => {
    void load();
  }, [load]);

  const live = tokens?.filter((token) => token.active) ?? [];
  const spent = (tokens?.length ?? 0) - live.length;

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
          onRevoked={() => void load()}
          onFailure={onFailure}
        />
      )}
      {spent > 0 ? (
        <p className={tokenStyles.gone}>
          {spent === 1
            ? "One more has already stopped working."
            : `${spent} more have already stopped working.`}
        </p>
      ) : null}
    </section>
  );
}
