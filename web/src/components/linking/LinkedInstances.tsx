"use client";

import { Plug, RotateCcw } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Said, Trouble } from "@/components/ui/field";
import { readJSON, refusalMessage } from "@/lib/answer";
import { browserFetch } from "@/lib/api/browser-mutation";
import { useAuth } from "@/lib/auth";
import {
  isInstanceList,
  type ManagedInstance,
  revoked,
} from "@/lib/instance-standing";
import { InstanceRow } from "./InstanceRow";

type Notice = { kind: "said" | "trouble"; message: string };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function LinkedInstances() {
  const { account } = useAuth();
  const [instances, setInstances] = useState<
    ManagedInstance[] | null | undefined
  >(undefined);
  const [loadTrouble, setLoadTrouble] = useState("");
  const [notice, setNotice] = useState<Notice | null>(null);
  const [revoking, setRevoking] = useState("");

  const load = useCallback(async () => {
    setLoadTrouble("");
    setInstances(undefined);
    try {
      const response = await fetch("/api/v1/instances", {
        cache: "no-store",
        credentials: "same-origin",
      });
      const answer: unknown = await readJSON(response);
      if (!response.ok) {
        setLoadTrouble(
          refusalMessage(answer, "We could not read your linked instances."),
        );
        setInstances(null);
        return;
      }
      if (!isInstanceList(answer)) {
        setLoadTrouble("Illarin returned an incomplete instance list.");
        setInstances(null);
        return;
      }
      setInstances(answer.items);
    } catch {
      setLoadTrouble(UNREACHABLE);
      setInstances(null);
    }
  }, []);

  useEffect(() => {
    if (account === undefined) return;
    if (!account) {
      setInstances([]);
      return;
    }
    void load();
  }, [account, load]);

  if (!account) return null;

  async function revoke(instance: ManagedInstance) {
    if (revoking) return;
    setNotice(null);
    setRevoking(instance.id);
    const named = `${instance.applicationName} — ${instance.instanceName}`;
    try {
      const response = await browserFetch(`/api/v1/instances/${instance.id}`, {
        credentials: "same-origin",
        method: "DELETE",
      });
      const answer: unknown = await readJSON(response);
      if (!response.ok) {
        setNotice({
          kind: "trouble",
          message: refusalMessage(answer, `${named} could not be revoked.`),
        });
        return;
      }

      const at = new Date().toISOString();
      setInstances(
        (current) =>
          current?.map((one) =>
            one.id === instance.id ? revoked(one, at) : one,
          ) ?? current,
      );
      setNotice({
        kind: "said",
        message: `${named} can no longer reach your account. Other linked instances were not changed.`,
      });
    } catch {
      setNotice({ kind: "trouble", message: UNREACHABLE });
    } finally {
      setRevoking("");
    }
  }

  return (
    <section aria-labelledby="linked-applications">
      <div className="flex flex-wrap items-end justify-between gap-x-8 gap-y-4">
        <div className="min-w-0 max-w-[52ch]">
          <h2
            className="font-display text-section font-medium tracking-tight text-ink"
            id="linked-applications"
          >
            Linked applications
          </h2>
          <p className="mt-2 font-prose text-ui text-mute">
            Each link connects one application installation to your account.
            Revoke a link to stop that installation's access.
          </p>
        </div>
        <Button asChild variant="secondary">
          <Link href="/link">Link an application</Link>
        </Button>
      </div>

      {notice ? (
        <div className="mt-5">
          {notice.kind === "trouble" ? (
            <Trouble>{notice.message}</Trouble>
          ) : (
            <Said>{notice.message}</Said>
          )}
        </div>
      ) : null}

      <div aria-busy={instances === undefined} className="mt-6">
        {instances === undefined ? (
          <p aria-live="polite" className="font-ui text-ui text-mute">
            Loading your linked applications…
          </p>
        ) : instances === null ? (
          <div className="flex flex-wrap items-center gap-4 rounded-plate bg-stop-wash px-5 py-4">
            <RotateCcw
              aria-hidden="true"
              className="size-5 shrink-0 text-stop"
              strokeWidth={1.6}
            />
            <p className="min-w-0 flex-1 basis-56 font-ui text-ui text-ink">
              {loadTrouble}
            </p>
            <Button onClick={() => void load()} variant="secondary">
              Try again
            </Button>
          </div>
        ) : instances.length === 0 ? (
          <div className="flex flex-wrap items-center gap-4 rounded-plate bg-deep px-6 py-6">
            <Plug
              aria-hidden="true"
              className="size-6 shrink-0 text-mute"
              strokeWidth={1.5}
            />
            <p className="min-w-0 flex-1 basis-64 font-prose text-ui text-mute">
              No applications linked. Start linking in your application, or
              enter the code it gives you.
            </p>
          </div>
        ) : (
          <ul className="m-0 grid list-none gap-px overflow-hidden rounded-plate bg-rule p-0">
            {instances.map((instance) => (
              <InstanceRow
                busy={Boolean(revoking)}
                instance={instance}
                key={instance.id}
                onRevoke={() => void revoke(instance)}
                revoking={revoking === instance.id}
              />
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
