"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Download, Send } from "lucide-react";
import { useCallback, useState } from "react";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api/client";
import {
  type DownloadFormat,
  fetchWork,
  type WorkConnectedApp,
  type WorkConnectedAppList,
  type WorkDetail,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import {
  downloadAddress,
  installsInApp,
  mainAction,
  orderedFormats,
  readerAppFormat,
  sendActionLabel,
} from "@/lib/work-send";

const appNames = new Intl.ListFormat("en-US", { type: "conjunction" });

function peekKey(id: string) {
  return ["works", "peek", id] as const;
}

/** usePeek reads a work's detail and the reader's connected apps once something in Browse asks to look inside it. */
export function usePeek(id: string, wanted: boolean) {
  const { account } = useAuth();
  const work = useQuery({
    queryKey: peekKey(id),
    queryFn: () => fetchWork(id),
    enabled: wanted,
    staleTime: 60_000,
  });
  const apps = useQuery({
    queryKey: [...peekKey(id), "apps", account?.handle],
    queryFn: async () => {
      const { data } = await api<WorkConnectedAppList>(
        "GET",
        `/v1/works/${id}/connected-apps`,
        { cache: "no-store" },
      );
      return data?.items ?? [];
    },
    enabled: wanted && Boolean(account),
    staleTime: 30_000,
  });
  return {
    work: work.data ?? null,
    failed: work.isError,
    apps: apps.data ?? [],
  };
}

/** usePrefetchPeek starts reading a work's detail when the pointer comes near the control that shows it. */
export function usePrefetchPeek() {
  const client = useQueryClient();
  return useCallback(
    (id: string) =>
      void client.prefetchQuery({
        queryKey: peekKey(id),
        queryFn: () => fetchWork(id),
        staleTime: 60_000,
      }),
    [client],
  );
}

function amount(role: DownloadFormat["roles"][number]): string {
  if (role.verdict === "dropped") return "Left out";
  if (role.verdict === "reduced") return role.reason ?? "In part";
  return role.sample.count > 1 ? role.sample.count.toLocaleString("en-US") : "";
}

/** heldFormat is the file a reader would get: the one for their app, else the recommended one. */
export function heldFormat(work: WorkDetail): DownloadFormat | null {
  const forApp = readerAppFormat(work.appFormats, work.readerApp);
  return orderedFormats(work.downloads, forApp?.format ?? null)[0] ?? null;
}

/** Holds lists what the reader's file of a work carries, one row per part. */
export function Holds({
  className,
  format,
}: {
  className?: string;
  format: DownloadFormat;
}) {
  return (
    <ul className={cn("m-0 flex list-none flex-col p-0", className)}>
      {format.roles.map((role) => (
        <li
          className="flex min-h-8 items-baseline justify-between gap-4 border-t border-rule/60 py-1.5 text-meta first:border-t-0"
          key={role.role}
        >
          <span
            className={role.verdict === "dropped" ? "text-mute" : "text-ink"}
          >
            {role.label}
          </span>
          <span className="text-mute tabular-nums">{amount(role)}</span>
        </li>
      ))}
    </ul>
  );
}

/** GetIt is Browse's one button for a work: send it to the reader's connected app, else download the file their app reads. */
export function GetIt({
  apps,
  className,
  work,
}: {
  apps: WorkConnectedApp[];
  className?: string;
  work: WorkDetail;
}) {
  const [state, setState] = useState<"idle" | "busy" | "done" | "failed">(
    "idle",
  );
  const forApp = readerAppFormat(work.appFormats, work.readerApp);
  const downloads = work.hasPrivatePrompts
    ? []
    : orderedFormats(work.downloads, forApp?.format ?? null);
  const main = mainAction({
    connected: apps,
    downloads,
    forApp,
    hasOriginal: false,
    readerApp: work.readerApp,
  });
  if (!main || main.kind === "original") return null;

  if (main.kind === "send") {
    return (
      <div className={cn("flex flex-col gap-1.5", className)}>
        <Button
          className="w-full"
          loading={state === "busy"}
          onClick={async () => {
            setState("busy");
            const { response } = await api<unknown>(
              "POST",
              `/v1/works/${work.id}/sends`,
              { body: { connectedAppId: main.app.connectedAppId } },
            ).catch(() => ({ response: { ok: false } }));
            setState(response.ok ? "done" : "failed");
          }}
          variant="primary"
        >
          {state === "done" ? (
            <Check aria-hidden="true" />
          ) : (
            <Send aria-hidden="true" />
          )}
          {state === "done"
            ? `Sent to ${main.app.name}`
            : sendActionLabel(main.app, installsInApp(work.type))}
        </Button>
        {state === "failed" ? (
          <p className="text-meta text-stop" role="alert">
            Illarin could not send it. Try again.
          </p>
        ) : null}
      </div>
    );
  }

  const readers = work.appFormats
    .filter((app) => app.format === main.format.format)
    .map((app) => app.label);
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <Button asChild className="w-full" variant="primary">
        <a
          href={downloadAddress({
            workId: work.id,
            format: main.format.format,
          })}
          onClick={() => setState("done")}
        >
          {state === "done" ? (
            <Check aria-hidden="true" />
          ) : (
            <Download aria-hidden="true" />
          )}
          {state === "done"
            ? "Downloaded"
            : main.label
              ? `Download for ${main.label}`
              : "Download"}
        </a>
      </Button>
      <p className="text-meta text-mute">
        {main.format.label}
        {readers.length ? ` · works in ${appNames.format(readers)}` : ""}
      </p>
    </div>
  );
}
