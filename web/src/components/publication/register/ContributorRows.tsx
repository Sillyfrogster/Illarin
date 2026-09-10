"use client";

import { UserRoundPlus } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import { Field, TextInput } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
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
import { readableDate } from "@/lib/dates";
import { grantAllowance, nothingIn } from "@/lib/publication-register";
import { GrantTokens } from "../GrantTokens";
import { CategoryChoice, DestinationChoice } from "./choices";
import {
  Nothing,
  PanelHead,
  Past,
  PastRow,
  Row,
  Rows,
  StartAction,
} from "./RowParts";
import { Consequence, StepForm } from "./StepParts";

/** Everyone Illarin has approved to publish, and what each approval covers. */
export function ContributorRows({
  apps,
  grants,
  onOpen,
}: {
  apps: PublicationApp[];
  grants: PublicationGrant[];
  onOpen: (grant: PublicationGrant | null) => void;
}) {
  const active = grants.filter((one) => one.active);
  const ended = grants.filter((one) => !one.active);
  const open = apps.filter((one) => !one.retired);

  return (
    <>
      <PanelHead
        action={
          <StartAction
            disabled={open.length === 0}
            icon={UserRoundPlus}
            onClick={() => onOpen(null)}
          >
            Approve someone
          </StartAction>
        }
        id="register-heading"
        title="Contributors"
      />

      {active.length === 0 ? (
        <Nothing>{nothingIn("contributors", { apps })}</Nothing>
      ) : (
        <Rows>
          {active.map((grant) => (
            <Row
              facts={
                <>
                  {grant.holder.displayName ? (
                    <span>@{grant.holder.handle}</span>
                  ) : null}
                  <span>{grantAllowance(grant)}</span>
                </>
              }
              key={grant.id}
              lead={
                <CreatorPortrait
                  handle={grant.holder.handle}
                  picture={grant.holder.avatar}
                  size="sm"
                />
              }
              onOpen={() => onOpen(grant)}
              open={`Change what @${grant.holder.handle} may publish`}
              title={grant.holder.displayName || `@${grant.holder.handle}`}
            />
          ))}
        </Rows>
      )}

      {ended.length > 0 ? (
        <Past summary={`${ended.length} ended`}>
          {ended.map((grant) => (
            <PastRow key={grant.id}>
              @{grant.holder.handle} for {grant.app.name}
              {grant.revokedAt ? `, ${readableDate(grant.revokedAt)}` : null}
            </PastRow>
          ))}
        </Past>
      ) : null}
    </>
  );
}

/** Approving somebody, or changing what somebody already approved may publish. */
export function ContributorStep({
  apps,
  categories,
  destinations,
  existing,
  onClose,
  onFailure,
  onRevoked,
  onSaved,
}: {
  apps: PublicationApp[];
  categories: PublicationCategory[];
  destinations: PublicationDestination[];
  existing: PublicationGrant | null;
  onClose: () => void;
  onFailure: (message: string) => void;
  onRevoked: () => void;
  onSaved: (saved: PublicationGrant, added: boolean) => void;
}) {
  const [handle, setHandle] = useState("");
  const [appId, setAppId] = useState("");
  const [allowed, setAllowed] = useState<string[]>(
    existing ? existing.categories.map((one) => one.id) : [],
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
  const [busy, setBusy] = useState(false);

  const open = apps.filter((one) => !one.retired);
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
          appId,
          categoryIds: allowed,
          defaultCategoryId: fallback,
          handle: handle.trim().replace(/^@/, ""),
        });
    if (written.error || !written.value) {
      setBusy(false);
      onFailure(written.error ?? "");
      return;
    }
    const policed = await setGrantDestinations(written.value.id, {
      defaultDestinationIds: follows ? [] : sends,
      destinationIds: follows ? null : reaching,
    });
    setBusy(false);
    if (policed.error || !policed.value) {
      onSaved(written.value, !existing);
      onFailure(policed.error ?? "");
      return;
    }
    onSaved(policed.value, !existing);
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
    <>
      {existing ? (
        <Link
          className="mb-5 inline-flex min-h-11 items-center font-ui text-meta font-medium text-accent underline-offset-4 outline-offset-3 hover:underline"
          href={`/@${existing.holder.handle}`}
        >
          Look at their profile
        </Link>
      ) : null}
      <StepForm
        busy={busy}
        commit={existing ? "Save" : "Approve"}
        onCommit={save}
        ready={ready}
        under={
          existing ? (
            <Consequence
              action="Revoke the approval"
              busy={busy}
              confirm="Revoke, and take the badge back"
              onConfirm={revoke}
            >
              Everything @{existing.holder.handle} published stays, under their
              name. Their tokens stop working at once.
            </Consequence>
          ) : null
        }
      >
        {existing ? null : (
          <>
            <Field
              hint="The account Illarin is approving. They keep publishing under their own name."
              htmlFor="approve-handle"
              label="Handle"
            >
              <TextInput
                autoComplete="off"
                id="approve-handle"
                onChange={(event) => setHandle(event.target.value)}
                placeholder="kestrel.writes"
                value={handle}
              />
            </Field>
            <Field
              hint="One approval covers one app. Changing it later needs a new approval."
              htmlFor="approve-app"
              label="App"
            >
              <Select
                className="w-full"
                id="approve-app"
                onChange={(event) => setAppId(event.target.value)}
                value={appId}
              >
                <option value="">Choose an app</option>
                {open.map((app) => (
                  <option key={app.id} value={app.id}>
                    {app.name}
                  </option>
                ))}
              </Select>
            </Field>
          </>
        )}

        <CategoryChoice
          allowed={allowed}
          categories={categories}
          fallback={fallback}
          name={existing?.id ?? "approve"}
          onAllowed={setAllowed}
          onFallback={setFallback}
        />

        <DestinationChoice
          allowed={reaching}
          defaults={sends}
          destinations={destinations}
          inherit={{ label: appPolicy(existing, apps, appId), on: follows }}
          legend="Where they may announce"
          onAllowed={setReaching}
          onDefaults={setSends}
          onInherit={setFollows}
        />
      </StepForm>

      {existing ? (
        <div className="mt-8 border-t border-rule/45 pt-6">
          <GrantTokens grant={existing} mine={false} />
        </div>
      ) : null}
    </>
  );
}

/** Says what following the app means here, naming the app rather than leaving it abstract. */
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
