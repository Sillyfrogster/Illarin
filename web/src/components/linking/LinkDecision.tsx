"use client";

import { Check, ShieldAlert } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { type PendingLink, readableExpiry } from "@/lib/link-request";
import { describeScope } from "@/lib/scopes";
import { DeclaredValues } from "./DeclaredValues";

export type Decision = "approve" | "deny";

export function LinkDecision({
  deciding,
  link,
  onCancel,
  onDecide,
  trouble,
  userCode,
}: {
  deciding: Decision | null;
  link: PendingLink;
  onCancel?: () => void;
  onDecide: (decision: Decision) => void;
  trouble: string;
  userCode?: string;
}) {
  const busy = deciding !== null;

  return (
    <div aria-busy={busy} className="grid gap-8">
      <div>
        <p className="inline-flex items-center gap-2 rounded-control bg-deep px-3 py-1.5 font-ui text-meta text-mute">
          <ShieldAlert
            aria-hidden="true"
            className="size-4 shrink-0"
            strokeWidth={1.7}
          />
          Unverified application
        </p>
        <h2 className="mt-4 font-display text-[clamp(1.7rem,3vw,2.6rem)] leading-[1.05] font-medium tracking-[-0.04em] text-ink [overflow-wrap:anywhere]">
          {link.applicationName}
        </h2>
        <dl className="mt-4 flex flex-wrap gap-x-8 gap-y-2 font-ui text-ui">
          <Fact label="Installation">
            <span className="[overflow-wrap:anywhere]">
              {link.instanceName}
            </span>
          </Fact>
          {link.applicationVersion ? (
            <Fact label="Version">{link.applicationVersion}</Fact>
          ) : null}
          <Fact label="Request expires">
            <time dateTime={link.expiresAt}>
              {readableExpiry(link.expiresAt)}
            </time>
          </Fact>
        </dl>
      </div>

      {userCode ? (
        <div className="flex items-start gap-4 rounded-plate bg-stop-wash px-5 py-4">
          <ShieldAlert
            aria-hidden="true"
            className="mt-0.5 size-5 shrink-0 text-stop"
            strokeWidth={1.7}
          />
          <p className="font-prose text-ui text-ink">
            <strong className="font-medium">
              Match this code before continuing.
            </strong>{" "}
            Confirm{" "}
            <b className="font-mono tracking-wider text-stop">{userCode}</b> is
            exactly what the application shows. If it differs, decline.
          </p>
        </div>
      ) : null}

      <section aria-labelledby="link-permissions">
        <h3
          className="font-ui text-ui font-medium text-ink"
          id="link-permissions"
        >
          What it would be able to do
        </h3>
        <ul className="m-0 mt-4 grid list-none gap-4 p-0">
          {link.scopes.map((scope) => {
            const copy = describeScope(scope);
            return (
              <li className="flex items-start gap-3.5" key={scope}>
                <span className="mt-0.5 grid size-6 shrink-0 place-items-center rounded-full bg-accent-wash text-accent">
                  <Check
                    aria-hidden="true"
                    className="size-3.5"
                    strokeWidth={2.4}
                  />
                </span>
                <div className="min-w-0">
                  <strong className="block font-ui text-ui font-medium text-ink">
                    {copy.title}
                  </strong>
                  <span className="font-prose text-ui text-mute">
                    {copy.detail}
                  </span>
                </div>
              </li>
            );
          })}
        </ul>
      </section>

      <p className="font-prose text-ui text-mute" id="unverified-link-details">
        These names and compatibility details were supplied by the application,
        not verified by Illarin. Approve only if you started this request.
      </p>

      {trouble ? <Trouble>{trouble}</Trouble> : null}

      <div
        aria-describedby="unverified-link-details"
        className="flex flex-wrap items-center gap-3"
      >
        <Button
          disabled={busy}
          loading={deciding === "approve"}
          onClick={() => onDecide("approve")}
          size="large"
          variant="primary"
        >
          {deciding === "approve" ? "Approving" : "Approve"}
        </Button>
        <Button
          disabled={busy}
          loading={deciding === "deny"}
          onClick={() => onDecide("deny")}
          size="large"
          variant="outline"
        >
          {deciding === "deny" ? "Declining" : "Decline"}
        </Button>
        {onCancel && !busy ? (
          <Button onClick={onCancel} variant="ghost">
            Use a different code
          </Button>
        ) : null}
      </div>

      <MorphingDisclosure
        className="border-t border-rule pt-3"
        summary="What this application says about itself"
      >
        <div className="pt-4 pb-2">
          <p className="font-ui text-meta text-mute">
            Self-reported technical details. They never grant permission.
          </p>
          <dl className="mt-4 grid gap-4 sm:grid-cols-3">
            <DeclaredValues
              label="Accepted targets"
              values={link.acceptedTargets}
            />
            <DeclaredValues label="Capabilities" values={link.capabilities} />
            <div>
              <dt className="font-ui text-meta text-mute">Protocol</dt>
              <dd className="mt-1 font-ui text-ui text-ink">
                Version {link.protocolVersion}
              </dd>
            </div>
          </dl>
        </div>
      </MorphingDisclosure>
    </div>
  );
}

function Fact({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="min-w-0">
      <dt className="font-ui text-meta text-mute">{label}</dt>
      <dd className="mt-0.5 font-ui text-ui text-ink">{children}</dd>
    </div>
  );
}
