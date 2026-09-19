"use client";

import { Plug, RotateCcw } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Said, Trouble } from "@/components/ui/field";
import { refusalMessage } from "@/lib/answer";
import { api } from "@/lib/api/client";
import { useAuth } from "@/lib/auth";
import {
  isConnectedAppList,
  type ManagedConnectedApp,
  revoked,
} from "@/lib/connected-app-standing";
import { ConnectedAppRow } from "./ConnectedAppRow";

type Notice = { kind: "said" | "trouble"; message: string };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function ConnectedApps() {
  const { account } = useAuth();
  const [apps, setApps] = useState<ManagedConnectedApp[] | null | undefined>(
    undefined,
  );
  const [loadTrouble, setLoadTrouble] = useState("");
  const [notice, setNotice] = useState<Notice | null>(null);
  const [revoking, setRevoking] = useState("");

  const load = useCallback(async () => {
    setLoadTrouble("");
    setApps(undefined);
    try {
      const { data, error, response } = await api<unknown>(
        "GET",
        "/v1/connected-apps",
        { cache: "no-store" },
      );
      const answer = response.ok ? data : error;
      if (!response.ok) {
        setLoadTrouble(
          refusalMessage(answer, "We could not read your connected apps."),
        );
        setApps(null);
        return;
      }
      if (!isConnectedAppList(answer)) {
        setLoadTrouble(
          "Illarin returned an incomplete list of connected apps.",
        );
        setApps(null);
        return;
      }
      setApps(answer.items);
    } catch {
      setLoadTrouble(UNREACHABLE);
      setApps(null);
    }
  }, []);

  useEffect(() => {
    if (account === undefined) return;
    if (!account) {
      setApps([]);
      return;
    }
    void load();
  }, [account, load]);

  if (!account) return null;

  async function revoke(app: ManagedConnectedApp) {
    if (revoking) return;
    setNotice(null);
    setRevoking(app.id);
    const named = `${app.appName} — ${app.name}`;
    try {
      const { error, response } = await api<void>(
        "DELETE",
        `/v1/connected-apps/${app.id}`,
      );
      const answer = error;
      if (!response.ok) {
        setNotice({
          kind: "trouble",
          message: refusalMessage(answer, `${named} could not be revoked.`),
        });
        return;
      }

      const at = new Date().toISOString();
      setApps(
        (current) =>
          current?.map((one) => (one.id === app.id ? revoked(one, at) : one)) ??
          current,
      );
      setNotice({
        kind: "said",
        message: `${named} can no longer reach your account. Your other connected apps were not changed.`,
      });
    } catch {
      setNotice({ kind: "trouble", message: UNREACHABLE });
    } finally {
      setRevoking("");
    }
  }

  return (
    <section aria-labelledby="connected-apps">
      <div className="flex flex-wrap items-end justify-between gap-x-8 gap-y-4">
        <div className="min-w-0 max-w-[52ch]">
          <h2
            className="font-display text-section font-medium tracking-tight text-ink"
            id="connected-apps"
          >
            Connected apps
          </h2>
          <p className="mt-2 font-prose text-ui text-mute">
            Each connected app is one copy of an app you approved. Revoke one to
            stop its access; the others keep theirs.
          </p>
        </div>
        <Button asChild variant="secondary">
          <Link href="/connect">Connect an app</Link>
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

      <div aria-busy={apps === undefined} className="mt-6">
        {apps === undefined ? (
          <p aria-live="polite" className="font-ui text-ui text-mute">
            Loading your connected apps…
          </p>
        ) : apps === null ? (
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
        ) : apps.length === 0 ? (
          <div className="flex flex-wrap items-center gap-4 rounded-plate bg-deep px-6 py-6">
            <Plug
              aria-hidden="true"
              className="size-6 shrink-0 text-mute"
              strokeWidth={1.5}
            />
            <p className="min-w-0 flex-1 basis-64 font-prose text-ui text-mute">
              No connected apps. Start connecting in your app, or enter the code
              it gives you.
            </p>
          </div>
        ) : (
          <ul className="m-0 grid list-none gap-px overflow-hidden rounded-plate bg-rule p-0">
            {apps.map((app) => (
              <ConnectedAppRow
                app={app}
                busy={Boolean(revoking)}
                key={app.id}
                onRevoke={() => void revoke(app)}
                revoking={revoking === app.id}
              />
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
