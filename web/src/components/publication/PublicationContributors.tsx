"use client";

import { Plus, UserRound } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { type FormEvent, useCallback, useEffect, useState } from "react";
import {
  approveContributor,
  readApps,
  readCategories,
  readGrants,
  revokeGrant,
  updateGrant,
} from "@/lib/api/publication";
import type {
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
} from "@/lib/api/query";
import { CategoryChoice } from "./CategoryChoice";
import styles from "./PublicationContributors.module.css";
import section from "./PublicationSection.module.css";

export function PublicationContributors() {
  const [apps, setApps] = useState<PublicationApp[]>([]);
  const [categories, setCategories] = useState<PublicationCategory[]>([]);
  const [grants, setGrants] = useState<PublicationGrant[] | null>(null);
  const [failure, setFailure] = useState("");
  const [handle, setHandle] = useState("");
  const [appId, setAppId] = useState("");
  const [allowed, setAllowed] = useState<string[]>([]);
  const [fallback, setFallback] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const [appAnswer, categoryAnswer, grantAnswer] = await Promise.all([
      readApps(),
      readCategories(),
      readGrants(),
    ]);
    const trouble =
      appAnswer.error ?? categoryAnswer.error ?? grantAnswer.error ?? "";
    if (trouble) {
      setFailure(trouble);
      return;
    }
    setFailure("");
    setApps(appAnswer.value?.apps ?? []);
    setCategories(categoryAnswer.value?.categories ?? []);
    setGrants(grantAnswer.value?.grants ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const open = apps.filter((app) => !app.retired);
  const active = (grants ?? []).filter((grant) => grant.active);
  const ended = (grants ?? []).filter((grant) => !grant.active);

  async function approve(event: FormEvent) {
    event.preventDefault();
    const wanted = handle.trim().replace(/^@/, "");
    if (!wanted || !appId || allowed.length === 0 || !fallback || busy) return;
    setBusy(true);
    const answer = await approveContributor({
      handle: wanted,
      appId,
      categoryIds: allowed,
      defaultCategoryId: fallback,
    });
    setBusy(false);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setHandle("");
    setAppId("");
    setAllowed([]);
    setFallback("");
    setGrants([answer.value, ...(grants ?? [])]);
  }

  async function narrow(
    grant: PublicationGrant,
    categoryIds: string[],
    defaultCategoryId: string,
  ) {
    const answer = await updateGrant(grant.id, {
      categoryIds,
      defaultCategoryId,
    });
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return false;
    }
    setFailure("");
    setGrants((known) =>
      (known ?? []).map((one) =>
        one.id === grant.id ? (answer.value as PublicationGrant) : one,
      ),
    );
    return true;
  }

  async function revoke(grant: PublicationGrant) {
    const answer = await revokeGrant(grant.id);
    if (answer.error) {
      setFailure(answer.error);
      return;
    }
    setFailure("");
    await load();
  }

  if (!grants) {
    return (
      <p className={section.loading} aria-live="polite">
        {failure || "Reading the approvals…"}
      </p>
    );
  }

  return (
    <div className={section.region}>
      {failure ? (
        <p className={section.failure} role="alert">
          {failure}
        </p>
      ) : null}

      {active.length > 0 ? (
        <ul className={styles.people}>
          {active.map((grant) => (
            <ContributorCard
              key={grant.id}
              grant={grant}
              categories={categories}
              onNarrow={narrow}
              onRevoke={revoke}
            />
          ))}
        </ul>
      ) : (
        <p className={section.empty}>Nobody is approved to publish yet.</p>
      )}

      <form className={section.add} onSubmit={approve}>
        <h3>Approve a contributor</h3>
        <div className={section.fields}>
          <label htmlFor="approve-handle">
            Handle
            <input
              id="approve-handle"
              value={handle}
              placeholder="kestrel.writes"
              onChange={(event) => setHandle(event.target.value)}
            />
          </label>
          <label htmlFor="approve-app">
            App
            <select
              className={styles.select}
              id="approve-app"
              value={appId}
              onChange={(event) => setAppId(event.target.value)}
            >
              <option value="">Choose an app</option>
              {open.map((app) => (
                <option key={app.id} value={app.id}>
                  {app.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        <CategoryChoice
          name="approve"
          categories={categories}
          allowed={allowed}
          fallback={fallback}
          onAllowed={setAllowed}
          onFallback={setFallback}
        />
        <p className={section.note}>
          They publish directly, under their own name and this app. They gain no
          moderation, catalogue or account authority.
        </p>
        <button
          type="submit"
          disabled={
            busy ||
            !handle.trim() ||
            !appId ||
            allowed.length === 0 ||
            !fallback
          }
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          {busy ? "Approving" : "Approve"}
        </button>
      </form>

      {ended.length > 0 ? (
        <div className={section.retired}>
          <h3>Ended</h3>
          <ul>
            {ended.map((grant) => (
              <li key={grant.id}>
                <span>
                  @{grant.holder.handle} for {grant.app.name}
                </span>
                <span className={styles.when}>
                  {grant.revokedAt ? shownDate(grant.revokedAt) : null}
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}

function ContributorCard({
  grant,
  categories,
  onNarrow,
  onRevoke,
}: {
  grant: PublicationGrant;
  categories: PublicationCategory[];
  onNarrow: (
    grant: PublicationGrant,
    categoryIds: string[],
    defaultCategoryId: string,
  ) => Promise<boolean>;
  onRevoke: (grant: PublicationGrant) => Promise<void>;
}) {
  const [changing, setChanging] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [allowed, setAllowed] = useState<string[]>([]);
  const [fallback, setFallback] = useState("");

  function startChanging() {
    setAllowed(grant.categories.map((category) => category.id));
    setFallback(grant.defaultCategory.id);
    setChanging(true);
  }

  async function save() {
    if (allowed.length === 0 || !fallback) return;
    if (await onNarrow(grant, allowed, fallback)) setChanging(false);
  }

  return (
    <li className={styles.person}>
      <span className={styles.portrait}>
        {grant.holder.avatar ? (
          <Image
            src={grant.holder.avatar.url}
            alt=""
            width={52}
            height={52}
            unoptimized
          />
        ) : (
          <UserRound size={22} strokeWidth={1.5} aria-hidden="true" />
        )}
      </span>

      <div className={styles.who}>
        {grant.holder.displayName ? (
          <>
            <p className={styles.said}>{grant.holder.displayName}</p>
            <Link className={styles.handle} href={`/${grant.holder.handle}`}>
              @{grant.holder.handle}
            </Link>
          </>
        ) : (
          <Link className={styles.said} href={`/${grant.holder.handle}`}>
            @{grant.holder.handle}
          </Link>
        )}
        <p className={styles.writesFor}>
          Publishes for <strong>{grant.app.name}</strong>
          {grant.app.retired ? " — that app is retired" : null}
        </p>
        {changing ? null : (
          <p className={styles.allowed}>
            {grant.categories.map((category) => category.label).join(", ")}.{" "}
            {grant.defaultCategory.label} by default.
          </p>
        )}
      </div>

      <div className={styles.decide}>
        {changing ? null : (
          <>
            <button
              type="button"
              className={section.textAction}
              onClick={startChanging}
            >
              Change categories
            </button>
            {confirming ? null : (
              <button
                type="button"
                className={styles.revoke}
                onClick={() => setConfirming(true)}
              >
                Revoke
              </button>
            )}
          </>
        )}
      </div>

      {changing ? (
        <div className={styles.editing}>
          <CategoryChoice
            name={grant.id}
            categories={categories}
            allowed={allowed}
            fallback={fallback}
            onAllowed={setAllowed}
            onFallback={setFallback}
          />
          <div className={styles.commit}>
            <button
              type="button"
              className={styles.keep}
              onClick={save}
              disabled={allowed.length === 0 || !fallback}
            >
              Save
            </button>
            <button
              type="button"
              className={section.textAction}
              onClick={() => setChanging(false)}
            >
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {confirming ? (
        <div className={styles.confirm}>
          <p>
            Revoke @{grant.holder.handle}? They lose the editor and the
            contributor badge at once. Everything they published stays, under
            their name.
          </p>
          <div className={styles.commit}>
            <button
              type="button"
              className={styles.revokeNow}
              onClick={() => {
                setConfirming(false);
                void onRevoke(grant);
              }}
            >
              Revoke
            </button>
            <button
              type="button"
              className={section.textAction}
              onClick={() => setConfirming(false)}
            >
              Keep it
            </button>
          </div>
        </div>
      ) : null}
    </li>
  );
}

function shownDate(when: string) {
  return new Date(when).toLocaleDateString(undefined, {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
