"use client";

import { Plug, PlugZap } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { cn } from "@/lib/cn";
import {
  installedHere,
  type ManagedConnectedApp,
  seenAt,
} from "@/lib/connected-app-standing";
import { readableDate } from "@/lib/dates";
import type { Permission } from "@/lib/permissions";
import { DeclaredValues } from "./DeclaredValues";
import { PermissionChoices } from "./PermissionChoices";

export function ConnectedAppRow({
  app,
  busy,
  onPermissions,
  onRevoke,
  revoking,
}: {
  app: ManagedConnectedApp;
  busy: boolean;
  onPermissions: (granted: Permission[]) => void;
  onRevoke: () => void;
  revoking: boolean;
}) {
  const [confirming, setConfirming] = useState(false);
  const cut = Boolean(app.revokedAt);
  const library = installedHere(app);
  const confirmationId = `revoke-${app.id}`;

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
            {app.appName}
          </h3>
          <p className="font-ui text-ui text-mute [overflow-wrap:anywhere]">
            {app.name}
          </p>
          <p className="mt-2 font-ui text-meta text-mute">
            {cut && app.revokedAt
              ? `Connected ${readableDate(app.connectedAt)}, revoked ${readableDate(app.revokedAt)}`
              : `Connected ${readableDate(app.connectedAt)} · ${seenAt(app)}`}
          </p>
          {library && !cut ? (
            <p className="font-ui text-meta text-mute">{library}</p>
          ) : null}
          {cut ? null : (
            <fieldset className="mt-4 max-w-[60ch]">
              <legend className="sr-only">Permissions</legend>
              <PermissionChoices
                disabled={busy}
                granted={app.permissions}
                onChange={onPermissions}
              />
            </fieldset>
          )}
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
                This connected app will lose access to your account.
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
              {revoking ? "Revoking" : confirming ? "Revoke access" : "Revoke"}
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
        <MorphingDisclosure className="mt-3" summary="Capabilities">
          <dl className="grid gap-4 pt-3 pb-1 sm:grid-cols-3">
            {app.appVersion ? (
              <DeclaredValues label="Version" values={[app.appVersion]} />
            ) : null}
            <DeclaredValues label="Formats" values={app.acceptedFormats} />
            <DeclaredValues label="Capabilities" values={app.capabilities} />
            {app.protocolVersion !== null ? (
              <DeclaredValues
                label="Protocol"
                values={[`Version ${app.protocolVersion}`]}
              />
            ) : null}
            <div className="sm:col-span-3">
              <dt className="font-ui text-meta text-mute">
                Refresh credential
              </dt>
              <dd className="mt-1 font-mono text-meta text-ink">
                {app.prefix}
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
