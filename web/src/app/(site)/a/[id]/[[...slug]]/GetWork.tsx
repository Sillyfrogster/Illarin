"use client";

import {
  Check,
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
import { createPortal } from "react-dom";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api/client";
import {
  type DownloadFormat,
  fetchRecordedVersionDownloads,
  type WorkConnectedApp,
  type WorkConnectedAppList,
  type WorkDetail,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { shortMoment } from "@/lib/dates";
import { installTrack } from "@/lib/install-track";
import { installedVersionsLine } from "@/lib/installed-app-versions";
import {
  downloadAddress,
  installsInApp,
  isWaiting,
  mainAction,
  orderedFormats,
  readerAppFormat,
  sendActionLabel,
  sendFailureLine,
} from "@/lib/work-send";
import { CompareFormats } from "./CompareFormats";
import { FileContents } from "./card/FileContents";
import { flyInto, lift } from "./card/fly";
import { useCardStage } from "./card/stage";
import { versionName } from "./card/WorkCard";
import { FollowOffer } from "./follow/FollowOffer";
import { InstallProgress } from "./InstallProgress";
import { useWorkspace } from "./workspace/state";

const POLL_INTERVAL_MS = 8000;
const POLL_LIMIT = 20;
const DOWNLOADED_MS = 2600;

const appNames = new Intl.ListFormat("en-US", { type: "conjunction" });

type Work = Pick<
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
  | "media"
>;

/** GetWork is the work page's one download button: it sends to a connected app, else downloads the file the work's apps read best, and turns the card over for the rest. */
export function GetWork({
  aside,
  connectedApps: initialApps,
  primary = true,
  sendable,
  typeLabel,
  work,
}: {
  aside?: ReactNode;
  connectedApps: WorkConnectedApp[];
  primary?: boolean;
  sendable: boolean;
  typeLabel: string;
  work: Work;
}) {
  const { account } = useAuth();
  const stage = useCardStage();
  const workspace = useWorkspace();
  const button = useRef<HTMLElement>(null);
  const [connectedApps, setConnectedApps] = useState(initialApps);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [offering, setOffering] = useState(false);
  const [comparing, setComparing] = useState(false);
  const [downloaded, setDownloaded] = useState(false);
  const polls = useRef(0);
  const installs = installsInApp(work.type);
  const viewing = primary ? stage.viewing : null;
  const older = useOlderDownload(work.id, viewing?.number ?? null);

  const read = useCallback(async () => {
    const { data } = await api<WorkConnectedAppList>(
      "GET",
      `/v1/works/${work.id}/connected-apps`,
      { cache: "no-store" },
    );
    setConnectedApps(data?.items ?? []);
  }, [work.id]);

  useEffect(() => {
    if (account === undefined) return;
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

  const { setCollecting } = stage;
  useEffect(() => {
    if (primary) setCollecting(waiting);
  }, [primary, waiting, setCollecting]);

  useEffect(() => {
    if (!downloaded) return;
    const timer = setTimeout(() => setDownloaded(false), DOWNLOADED_MS);
    return () => clearTimeout(timer);
  }, [downloaded]);

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
      setFailure("Can't reach Illarin. Check your connection and try again.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  const picture = work.media.find((image) => image.isCover)?.thumbUrl;

  function takeOff() {
    lift(stage.card.current);
    flyInto(stage.card.current, button.current, picture);
  }

  function downloadStarted() {
    takeOff();
    setDownloaded(true);
    setOffering(true);
  }

  async function send(app: WorkConnectedApp) {
    takeOff();
    const sent = await act(`/v1/works/${work.id}/sends`, "POST", {
      connectedAppId: app.connectedAppId,
    });
    if (sent) setOffering(true);
  }

  const forApp = readerAppFormat(work.appFormats, work.readerApp);
  const formats = orderedFormats(work.downloads, forApp?.format ?? null);
  const downloads = work.hasPrivatePrompts ? [] : formats;
  const receiving = connectedApps.filter((one) => one.canReceive);
  const original = work.isOwner ? work.original : null;
  const versionsLine = installedVersionsLine(work);
  const tracks = installs
    ? connectedApps.map(installTrack).filter((track) => track !== null)
    : [];
  const standing = installs
    ? []
    : connectedApps.filter((one) => one.send !== null);
  const main = mainAction({
    connected: connectedApps,
    downloads,
    forApp,
    hasOriginal: original !== null,
    readerApp: work.readerApp,
  });

  const mainFormat = main?.kind === "download" ? main.format : null;
  const backFormat = mainFormat ?? downloads[0] ?? null;
  const otherSends = receiving.filter(
    (app) => main?.kind !== "send" || app !== main.app,
  );
  const sends =
    otherSends.length > 0
      ? otherSends.map((app) => (
          <Button
            disabled={busy || isWaiting(app.send)}
            key={app.connectedAppId}
            onClick={() => void send(app)}
            size="compact"
            variant="ghost"
            className="-ml-3"
          >
            {isWaiting(app.send) ? (
              <Clock aria-hidden="true" />
            ) : (
              <Send aria-hidden="true" />
            )}
            {sendActionLabel(app, installs)}
          </Button>
        ))
      : null;
  const back =
    primary &&
    !workspace.editing &&
    stage.backSlot &&
    (backFormat || original || sends)
      ? createPortal(
          <FileContents
            main={backFormat}
            onCompare={downloads.length > 0 ? () => setComparing(true) : null}
            onDownload={() => setOffering(true)}
            original={original}
            others={downloads.filter((one) => one !== backFormat)}
            sends={sends}
            workId={work.id}
          />,
          stage.backSlot,
        )
      : null;

  if (!main) {
    return (
      <>
        {aside ? <div className="flex">{aside}</div> : null}
        {back}
      </>
    );
  }

  const readers = mainFormat
    ? work.appFormats
        .filter((app) => app.format === mainFormat.format)
        .map((app) => app.label)
    : [];

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        {viewing && !work.hasPrivatePrompts ? (
          <OlderButton
            button={button}
            older={older}
            onDownload={downloadStarted}
            version={versionName(viewing)}
            workId={work.id}
          />
        ) : main.kind === "send" ? (
          <>
            <SendButton
              app={main.app}
              busy={busy}
              button={button}
              installs={installs}
              onSend={send}
            />
            {backFormat ? (
              <Button asChild>
                <a
                  href={downloadAddress({
                    workId: work.id,
                    format: backFormat.format,
                  })}
                  onClick={() => setOffering(true)}
                >
                  <Download aria-hidden="true" />
                  Download
                </a>
              </Button>
            ) : null}
          </>
        ) : (
          <Button
            asChild
            className="min-w-0 flex-1 sm:max-w-80"
            ref={button as React.Ref<HTMLButtonElement>}
            variant="primary"
          >
            <a
              href={
                main.kind === "download"
                  ? downloadAddress({
                      workId: work.id,
                      format: main.format.format,
                    })
                  : `/download/${work.id}`
              }
              onClick={downloadStarted}
            >
              {downloaded ? (
                <Check aria-hidden="true" />
              ) : main.kind === "download" ? (
                <Download aria-hidden="true" />
              ) : (
                <FileDown aria-hidden="true" />
              )}
              {downloaded
                ? "Downloaded"
                : main.kind === "download"
                  ? main.label
                    ? `Download for ${main.label}`
                    : "Download"
                  : "Original upload"}
            </a>
          </Button>
        )}
        {aside}
      </div>
      <p className="mt-2.5 text-meta text-mute">
        {viewing
          ? "An older version, written from what it recorded then."
          : main.kind === "send"
            ? `Goes to ${main.app.name}, your ${main.app.appName} app.`
            : mainFormat
              ? `${mainFormat.label}${readers.length > 0 ? ` · works in ${appNames.format(readers)}` : ""}`
              : null}
      </p>
      {downloads.length > 0 ? (
        <CompareFormats
          onOpenChange={setComparing}
          open={comparing}
          type={work.type}
          typeLabel={typeLabel}
        />
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
      {back}
    </>
  );
}

type Older =
  | { state: "none" }
  | { state: "reading" }
  | { state: "ready"; format: DownloadFormat; number: number }
  | { state: "refused"; refusal: string };

/** useOlderDownload loads the file an older version downloads as, once the reader moves to it. */
function useOlderDownload(workId: string, number: number | null): Older {
  const [older, setOlder] = useState<Older>({ state: "none" });
  useEffect(() => {
    if (number === null) {
      setOlder({ state: "none" });
      return;
    }
    let live = true;
    setOlder({ state: "reading" });
    fetchRecordedVersionDownloads(workId, number).then((answer) => {
      if (!live) return;
      const format = answer.offered
        ? orderedFormats(answer.offered.downloads)[0]
        : undefined;
      setOlder(
        format
          ? { state: "ready", format, number }
          : {
              state: "refused",
              refusal:
                ("refusal" in answer && answer.refusal) ||
                "This version cannot be written in any format Illarin offers.",
            },
      );
    });
    return () => {
      live = false;
    };
  }, [workId, number]);
  return older;
}

function OlderButton({
  button,
  older,
  onDownload,
  version,
  workId,
}: {
  button: React.RefObject<HTMLElement | null>;
  older: Older;
  onDownload: () => void;
  version: string;
  workId: string;
}) {
  if (older.state === "refused") {
    return (
      <p className="flex items-start gap-2 text-meta text-mute">
        <CircleAlert aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        {older.refusal}
      </p>
    );
  }
  if (older.state !== "ready") {
    return (
      <Button className="min-w-0 flex-1 sm:max-w-80" loading variant="primary">
        Download {version}
      </Button>
    );
  }
  return (
    <Button
      asChild
      className="min-w-0 flex-1 sm:max-w-80"
      ref={button as React.Ref<HTMLButtonElement>}
      variant="primary"
    >
      <a
        href={downloadAddress({
          workId,
          format: older.format.format,
          version: older.number,
        })}
        onClick={onDownload}
      >
        <Download aria-hidden="true" />
        Download {version}
      </a>
    </Button>
  );
}

function SendButton({
  app,
  busy,
  button,
  installs,
  onSend,
}: {
  app: WorkConnectedApp;
  busy: boolean;
  button: React.RefObject<HTMLElement | null>;
  installs: boolean;
  onSend: (app: WorkConnectedApp) => Promise<void>;
}) {
  const pending = isWaiting(app.send);
  return (
    <Button
      className="min-w-0 flex-1 sm:max-w-80"
      disabled={pending}
      loading={busy}
      onClick={() => void onSend(app)}
      ref={button as React.Ref<HTMLButtonElement>}
      variant="primary"
    >
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
