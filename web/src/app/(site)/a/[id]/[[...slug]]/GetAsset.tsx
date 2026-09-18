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
import {
  formatChoices,
  installsOnInstance,
  isWaiting,
} from "@/lib/asset-delivery";
import { useAuth } from "@/lib/auth";
import { installTrack } from "@/lib/install-track";
import { installedVersionsLine } from "@/lib/installed-app-versions";
import { AssetChooser, type AssetChooserProps } from "./AssetChooser";
import { InstallProgress } from "./InstallProgress";
import { WatchOffer } from "./watch/WatchOffer";

const WATCH_INTERVAL_MS = 8000;
const WATCH_LIMIT = 20;

export function GetAsset({
  aside,
  installedAppVersions = [],
  sendable,
  ...props
}: AssetChooserProps & {
  aside?: ReactNode;
  installedAppVersions?: string[];
  sendable: boolean;
}) {
  const {
    assetId,
    kind,
    kindLabel,
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
  const watched = useRef(0);
  const installs = installsOnInstance(kind);

  const read = useCallback(async () => {
    const { data } = await api<WorkInstanceList>(
      "GET",
      `/v1/assets/${assetId}/instances`,
      { cache: "no-store" },
    );
    if (!data) {
      setInstances([]);
      return;
    }
    setInstances(data.items);
  }, [assetId]);

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
      watched.current = 0;
      return;
    }
    const timer = setInterval(() => {
      watched.current += 1;
      if (watched.current > WATCH_LIMIT) {
        clearInterval(timer);
        return;
      }
      void read();
    }, WATCH_INTERVAL_MS);
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
              watched.current = 0;
            }
          }}
          open={open}
        >
          <PopoverTrigger asChild>
            <Button className="min-w-52 justify-between" variant="primary">
              {installs && receiving ? "Install" : "Download"} {kindLabel}
              <ChevronDown aria-hidden="true" />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            aria-label={`${installs && receiving ? "Install" : "Download"} this ${kindLabel}`}
          >
            <AssetChooser
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
      {sentAway ? <WatchOffer /> : null}
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
