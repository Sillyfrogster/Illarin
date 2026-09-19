"use client";

import { ChevronDown } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { api } from "@/lib/api/client";
import type { WorkConnectedApp, WorkConnectedAppList } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { installTrack } from "@/lib/install-track";
import { installedVersionsLine } from "@/lib/installed-app-versions";
import { formatChoices, installsInApp, isWaiting } from "@/lib/work-send";
import { FollowOffer } from "./follow/FollowOffer";
import { InstallProgress } from "./InstallProgress";
import { WorkChooser, type WorkChooserProps } from "./WorkChooser";

const POLL_INTERVAL_MS = 8000;
const POLL_LIMIT = 20;

export function GetWork({
  aside,
  installedAppVersions = [],
  sendable,
  ...props
}: WorkChooserProps & {
  aside?: ReactNode;
  installedAppVersions?: string[];
  sendable: boolean;
}) {
  const {
    workId,
    type,
    typeLabel,
    downloads,
    appFormats,
    original,
    holdsNothing,
    linkedInstallOnly,
  } = props;
  const { account } = useAuth();
  const [connectedApps, setConnectedApps] = useState<WorkConnectedApp[]>([]);
  const [open, setOpen] = useState(false);
  const [opened, setOpened] = useState(0);
  const [busy, setBusy] = useState(false);
  const [sentAway, setSentAway] = useState(false);
  const polls = useRef(0);
  const installs = installsInApp(type);

  const read = useCallback(async () => {
    const { data } = await api<WorkConnectedAppList>(
      "GET",
      `/v1/works/${workId}/connected-apps`,
      { cache: "no-store" },
    );
    if (!data) {
      setConnectedApps([]);
      return;
    }
    setConnectedApps(data.items);
  }, [workId]);

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

  async function dismiss(sendId: string) {
    setBusy(true);
    try {
      await api<void>("DELETE", `/v1/sends/${sendId}`);
      await read();
    } finally {
      setBusy(false);
    }
  }

  const choices = formatChoices({
    downloads,
    holdsNothing,
    app: appFormats[0]?.id ?? "",
    apps: appFormats,
  });
  const receiving = connectedApps.some((one) => one.canReceive);
  const chooses = linkedInstallOnly
    ? receiving
    : choices.length > 0 || Boolean(original);
  if (!chooses) return aside ? <div className="flex">{aside}</div> : null;

  const tracks = installs
    ? connectedApps.map(installTrack).filter((track) => track !== null)
    : [];
  const versionsLine = installedVersionsLine({
    appFormats,
    installedAppVersions,
  });

  return (
    <>
      <div className="flex items-center gap-2">
        <Popover
          onOpenChange={(next) => {
            setOpen(next);
            if (next) {
              setOpened((count) => count + 1);
              polls.current = 0;
            }
          }}
          open={open}
        >
          <PopoverTrigger asChild>
            <Button className="min-w-52 justify-between" variant="primary">
              {installs && receiving ? "Install" : "Download"} {typeLabel}
              <ChevronDown aria-hidden="true" />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            aria-label={`${installs && receiving ? "Install" : "Download"} this ${typeLabel}`}
          >
            <WorkChooser
              key={opened}
              {...props}
              connectedApps={connectedApps}
              onSent={
                installs
                  ? () => {
                      setOpen(false);
                      setSentAway(true);
                    }
                  : undefined
              }
              refresh={read}
            />
          </PopoverContent>
        </Popover>
        {aside}
      </div>
      {sentAway ? <FollowOffer /> : null}
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
                  void dismiss(track.app.send.id);
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
