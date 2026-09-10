"use client";

import { Plug, PlugZap } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { cn } from "@/lib/cn";
import { readableDate } from "@/lib/dates";
import {
  installedHere,
  type ManagedInstance,
  seenAt,
} from "@/lib/instance-standing";
import { describeScope } from "@/lib/scopes";
import { DeclaredValues } from "./DeclaredValues";

/**
 * One installation, as much as it takes to tell it from another. What it
 * declared about itself waits behind a disclosure, because the reader is here
 * to recognise an installation and cut it off, not to read its capabilities.
 */
export function InstanceRow({
  busy,
  instance,
  onRevoke,
  revoking,
}: {
  /** True while any row is being revoked, so two cannot run at once. */
  busy: boolean;
  instance: ManagedInstance;
  onRevoke: () => void;
  revoking: boolean;
}) {
  const [confirming, setConfirming] = useState(false);
  const cut = Boolean(instance.revokedAt);
  const library = installedHere(instance);
  const confirmationId = `revoke-${instance.id}`;

  return (
    <li className={cn("bg-plane px-5 py-5", cut && "opacity-60")}>
      <div className="flex flex-wrap items-start gap-x-5 gap-y-4">
        <span
          className={cn(
            "grid size-11 shrink-0 place-items-center rounded-control",
            cut ? "bg-deep text-mute" : "bg-accent-wash text-accent",
          )}
        >
          {cut ? (
            <PlugZap aria-hidden="true" className="size-5" strokeWidth={1.5} />
          ) : (
            <Plug aria-hidden="true" className="size-5" strokeWidth={1.5} />
          )}
        </span>

        <div className="min-w-0 flex-1 basis-64">
          <h3 className="font-ui text-ui font-medium text-ink [overflow-wrap:anywhere]">
            {instance.applicationName}
          </h3>
          <p className="font-ui text-ui text-mute [overflow-wrap:anywhere]">
            {instance.instanceName}
          </p>
          <p className="mt-2 font-ui text-meta text-mute">
            {cut && instance.revokedAt
              ? `Linked ${readableDate(instance.linkedAt)}, revoked ${readableDate(instance.revokedAt)}`
              : `Linked ${readableDate(instance.linkedAt)} · ${seenAt(instance)}`}
          </p>
          {library && !cut ? (
            <p className="font-ui text-meta text-mute">{library}</p>
          ) : null}
          <ul
            aria-label="Granted permissions"
            className="m-0 mt-3 flex list-none flex-wrap gap-1.5 p-0"
          >
            {instance.scopes.map((scope) => {
              const copy = describeScope(scope);
              return (
                <li
                  className="rounded-control bg-deep px-2.5 py-1 font-ui text-meta text-mute"
                  key={scope}
                  title={copy.detail}
                >
                  {copy.title}
                </li>
              );
            })}
          </ul>
        </div>

        {cut ? (
          <span className="inline-flex min-h-11 items-center font-ui text-meta font-medium text-mute">
            Revoked
          </span>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            {confirming ? (
              <p
                className="basis-full font-ui text-meta text-mute sm:basis-auto"
                id={confirmationId}
              >
                This cuts off only this installation.
              </p>
            ) : null}
            <Button
              aria-describedby={confirming ? confirmationId : undefined}
              aria-expanded={confirming}
              disabled={busy && !revoking}
              loading={revoking}
              onClick={() => {
                if (confirming) onRevoke();
                else setConfirming(true);
              }}
              variant={confirming ? "stop" : "secondary"}
            >
              {revoking ? "Revoking" : confirming ? "Confirm revoke" : "Revoke"}
            </Button>
            {confirming ? (
              <Button
                disabled={busy}
                onClick={() => setConfirming(false)}
                variant="ghost"
              >
                Cancel
              </Button>
            ) : null}
          </div>
        )}
      </div>

      {cut ? null : (
        <MorphingDisclosure
          className="mt-3"
          summary="What this installation says about itself"
        >
          <dl className="grid gap-4 pt-3 pb-1 sm:grid-cols-3">
            {instance.applicationVersion ? (
              <DeclaredValues
                label="Version"
                values={[instance.applicationVersion]}
              />
            ) : null}
            <DeclaredValues label="Targets" values={instance.acceptedTargets} />
            <DeclaredValues
              label="Capabilities"
              values={instance.capabilities}
            />
            {instance.protocolVersion !== null ? (
              <DeclaredValues
                label="Protocol"
                values={[`Version ${instance.protocolVersion}`]}
              />
            ) : null}
            <div className="sm:col-span-3">
              <dt className="font-ui text-meta text-mute">
                Refresh credential
              </dt>
              <dd className="mt-1 font-mono text-meta text-ink">
                {instance.prefix}
              </dd>
            </div>
          </dl>
          <p className="pb-1 font-ui text-meta text-mute">
            Self-reported compatibility, not permission.
          </p>
        </MorphingDisclosure>
      )}
    </li>
  );
}
