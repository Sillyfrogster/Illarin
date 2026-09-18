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
import type { WorkInstance, WorkInstanceList } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { installTrack } from "@/lib/install-track";
import { installedVersionsLine } from "@/lib/installed-app-versions";
import {
  formatChoices,
  installsOnInstance,
  isWaiting,
} from "@/lib/work-delivery";
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
    appTargets,
    original,
    holdsNothing,
    linkedInstallOnly,
  } = props;
  const { account } = useAuth();
  const [instances, setInstances] = useState<WorkInstance[]>([]);
  const [open, setOpen] = useState(false);
  const [opened, setOpened] = useState(0);
  const [busy, setBusy] = useState(false);
  const [sentAway, setSentAway] = useState(false);
  const polls = useRef(0);
  const installs = installsOnInstance(type);

  const read = useCallback(async () => {
    const { data } = await api<WorkInstanceList>(
      "GET",
      `/v1/works/${workId}/instances`,
      { cache: "no-store" },
    );
    if (!data) {
      setInstances([]);
      return;
    }
    setInstances(data.items);
  }, [workId]);

  useEffect(() => {
    if (!account || !sendable) {
      setInstances([]);
      return;
    }
    void read();
  }, [account, sendable, read]);

  const waiting = instances.some((one) => isWaiting(one.delivery));
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

  async function dismiss(deliveryId: string) {
    setBusy(true);
    try {
      await api<void>("DELETE", `/v1/deliveries/${deliveryId}`);
      await read();
    } finally {
      setBusy(false);
    }
  }

  const choices = formatChoices({
    downloads,
    holdsNothing,
    app: appTargets[0]?.id ?? "",
    apps: appTargets,
  });
  const receiving = instances.some((one) => one.canReceive);
  const chooses = linkedInstallOnly
    ? receiving
    : choices.length > 0 || Boolean(original);
  if (!chooses) return aside ? <div className="flex">{aside}</div> : null;

  const tracks = installs
    ? instances.map(installTrack).filter((track) => track !== null)
    : [];
  const versionsLine = installedVersionsLine({
    appTargets,
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
              instances={instances}
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
              key={track.instance.instanceId}
              onDismiss={() => {
                if (track.instance.delivery) {
                  void dismiss(track.instance.delivery.id);
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
