"use client";

import {
  ChevronDown,
  CircleAlert,
  Clock,
  Download,
  FileDown,
  Send,
} from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { api } from "@/lib/api/client";
import type {
  WorkConnectedApp,
  WorkConnectedAppList,
  WorkDetail,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { shortMoment } from "@/lib/dates";
import { installTrack } from "@/lib/install-track";
import { installedVersionsLine } from "@/lib/installed-app-versions";
import {
  installsInApp,
  isWaiting,
  orderedFormats,
  readerAppFormat,
  sendActionLabel,
  sendFailureLine,
} from "@/lib/work-send";
import { CompareFormats } from "./CompareFormats";
import { FormatMenu } from "./FormatMenu";
import { FollowOffer } from "./follow/FollowOffer";
import { InstallProgress } from "./InstallProgress";

const POLL_INTERVAL_MS = 8000;
const POLL_LIMIT = 20;

/** GetWork is the work page's download control. */
export function GetWork({
  aside,
  sendable,
  typeLabel,
  work,
}: {
  aside?: ReactNode;
  sendable: boolean;
  typeLabel: string;
  work: Pick<
    WorkDetail,
    | "id"
    | "type"
    | "downloads"
    | "appFormats"
    | "readerApp"
    | "original"
    | "isOwner"
    | "hasPrivatePrompts"
    | "installedAppVersions"
  >;
}) {
  const { account } = useAuth();
  const [connectedApps, setConnectedApps] = useState<WorkConnectedApp[]>([]);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [offering, setOffering] = useState(false);
  const polls = useRef(0);
  const installs = installsInApp(work.type);

  const read = useCallback(async () => {
    const { data } = await api<WorkConnectedAppList>(
      "GET",
      `/v1/works/${work.id}/connected-apps`,
      { cache: "no-store" },
    );
    setConnectedApps(data?.items ?? []);
  }, [work.id]);

  useEffect(() => {
    if (!account || !sendable) {
      setConnectedApps([]);
      return;
    }
    void read();
  }, [account, sendable, read]);

  const waiting = connectedApps.some((one) => isWaiting(one.send));
  useEffect(() => {
    if (!waiting) {
      polls.current = 0;
      return;
    }
    const timer = setInterval(() => {
      polls.current += 1;
      if (polls.current > POLL_LIMIT) {
        clearInterval(timer);
        return;
      }
      void read();
    }, POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [waiting, read]);

  async function act(path: string, method: "POST" | "DELETE", body?: unknown) {
    setBusy(true);
    setFailure("");
    try {
      const { error, response } = await api<unknown>(method, path, { body });
      if (!response.ok) {
        setFailure(
          typeof error === "object" &&
            error !== null &&
            "error" in error &&
            typeof error.error === "string"
            ? error.error
            : "Illarin could not do that. Try again in a moment.",
        );
        return false;
      }
      await read();
      return true;
    } catch {
      setFailure(
        "We could not reach Illarin. Check your connection and try again.",
      );
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function send(app: WorkConnectedApp) {
    const sent = await act(`/v1/works/${work.id}/sends`, "POST", {
      connectedAppId: app.connectedAppId,
    });
    if (sent) setOffering(true);
  }

  const forApp = readerAppFormat(work.appFormats, work.readerApp);
  const formats = orderedFormats(work.downloads, forApp?.format ?? null);
  const downloads = work.hasPrivatePrompts ? [] : formats;
  const others = forApp
    ? downloads.filter((one) => one.format !== forApp.format)
    : [];
  const receiving = connectedApps.filter((one) => one.canReceive);
  const original = work.isOwner ? work.original : null;
  const versionsLine = installedVersionsLine(work);
  const tracks = installs
    ? connectedApps.map(installTrack).filter((track) => track !== null)
    : [];
  const standing = installs
    ? []
    : connectedApps.filter((one) => one.send !== null);

  if (downloads.length === 0 && !original && receiving.length === 0) {
    return aside ? <div className="flex">{aside}</div> : null;
  }

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        {downloads.length > 0 && forApp ? (
          <Button asChild variant="primary">
            <a
              href={`/download/${work.id}/${forApp.format}`}
              onClick={() => setOffering(true)}
            >
              <Download aria-hidden="true" />
              Download for {forApp.label}
            </a>
          </Button>
        ) : downloads.length > 0 ? (
          <FormatMenu
            downloads={downloads}
            onDownload={() => setOffering(true)}
            original={original}
            workId={work.id}
          >
            <Button variant="primary">
              <Download aria-hidden="true" />
              Download {typeLabel}
              <ChevronDown aria-hidden="true" />
            </Button>
          </FormatMenu>
        ) : null}
        {others.length > 0 ? (
          <FormatMenu
            downloads={others}
            onDownload={() => setOffering(true)}
            original={original}
            workId={work.id}
          >
            <Button>
              Other formats
              <ChevronDown aria-hidden="true" />
            </Button>
          </FormatMenu>
        ) : original && (forApp || downloads.length === 0) ? (
          <Button asChild>
            <a href={`/download/${work.id}`}>
              <FileDown aria-hidden="true" />
              Original upload
            </a>
          </Button>
        ) : null}
        {receiving.length === 1 ? (
          <SendButton
            app={receiving[0]}
            busy={busy}
            installs={installs}
            onSend={send}
          />
        ) : receiving.length > 1 ? (
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button disabled={busy}>
                <Send aria-hidden="true" />
                {installs ? "Install on" : "Send to"}
                <ChevronDown aria-hidden="true" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              {receiving.map((app) => (
                <DropdownMenuItem
                  disabled={isWaiting(app.send)}
                  key={app.connectedAppId}
                  onSelect={() => void send(app)}
                >
                  {sendActionLabel(app, installs)}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </div>
      {downloads.length > 0 || aside ? (
        <div className="mt-2 flex items-center gap-4">
          {downloads.length > 0 ? (
            <CompareFormats type={work.type} typeLabel={typeLabel} />
          ) : null}
          {aside}
        </div>
      ) : null}
      {failure ? (
        <output aria-live="polite" className="mt-3 block text-meta text-stop">
          {failure}
        </output>
      ) : null}
      {standing.map((app) => (
        <SendStanding
          app={app}
          busy={busy}
          key={app.connectedAppId}
          onDismiss={() =>
            app.send && void act(`/v1/sends/${app.send.id}`, "DELETE")
          }
        />
      ))}
      {offering ? <FollowOffer /> : null}
      {versionsLine ? (
        <p className="mt-4 max-w-[42ch] text-meta break-words text-mute">
          {versionsLine}
        </p>
      ) : null}
      {tracks.length > 0 ? (
        <div className="mt-5 flex flex-col gap-5">
          {tracks.map((track) => (
            <InstallProgress
              busy={busy}
              key={track.app.connectedAppId}
              onDismiss={() => {
                if (track.app.send) {
                  void act(`/v1/sends/${track.app.send.id}`, "DELETE");
                }
              }}
              track={track}
            />
          ))}
        </div>
      ) : null}
    </>
  );
}

function SendButton({
  app,
  busy,
  installs,
  onSend,
}: {
  app: WorkConnectedApp;
  busy: boolean;
  installs: boolean;
  onSend: (app: WorkConnectedApp) => Promise<void>;
}) {
  const pending = isWaiting(app.send);
  return (
    <Button disabled={pending} loading={busy} onClick={() => void onSend(app)}>
      {pending ? <Clock aria-hidden="true" /> : <Send aria-hidden="true" />}
      {sendActionLabel(app, installs)}
    </Button>
  );
}

/** SendStanding says where a send to one connected app got to. */
function SendStanding({
  app,
  busy,
  onDismiss,
}: {
  app: WorkConnectedApp;
  busy: boolean;
  onDismiss: () => void;
}) {
  const send = app.send;
  if (!send) return null;
  const failed = send.state === "failed";
  return (
    <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1">
      <p
        className={`flex items-start gap-2 text-meta ${failed ? "text-stop" : "text-mute"}`}
      >
        {isWaiting(send) ? (
          <Clock aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        ) : failed ? (
          <CircleAlert
            aria-hidden="true"
            className="mt-0.5 size-3.5 shrink-0"
          />
        ) : null}
        {isWaiting(send)
          ? `Waiting for ${app.name} to collect it.`
          : failed
            ? sendFailureLine(send.reason)
            : `Delivered to ${app.name}${send.settledAt ? ` ${shortMoment(send.settledAt)}` : ""}.`}
      </p>
      <Button
        disabled={busy}
        onClick={onDismiss}
        size="compact"
        variant="ghost"
      >
        {isWaiting(send) ? "Cancel" : "Dismiss"}
      </Button>
    </div>
  );
}
