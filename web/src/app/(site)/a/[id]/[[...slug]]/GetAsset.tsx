"use client";

import { ChevronDown } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { AssetInstance, AssetInstanceList } from "@/lib/api/query";
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

const WATCH_INTERVAL_MS = 8000;
const WATCH_LIMIT = 20;

export function GetAsset({
  installedAppVersions = [],
  ...props
}: AssetChooserProps & { installedAppVersions?: string[] }) {
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
  const [instances, setInstances] = useState<AssetInstance[]>([]);
  const [open, setOpen] = useState(false);
  const [opened, setOpened] = useState(0);
  const [busy, setBusy] = useState(false);
  const watched = useRef(0);
  const installs = installsOnInstance(kind);

  const read = useCallback(async () => {
    const response = await fetch(`/api/v1/assets/${assetId}/instances`, {
      cache: "no-store",
      credentials: "same-origin",
    });
    if (!response.ok) {
      setInstances([]);
      return;
    }
    const answer = (await response.json()) as AssetInstanceList;
    setInstances(answer.items);
  }, [assetId]);

  useEffect(() => {
    if (!account) {
      setInstances([]);
      return;
    }
    void read();
  }, [account, read]);

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
      await browserFetch(`/api/v1/deliveries/${deliveryId}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
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
  if (linkedInstallOnly && !receiving) return null;
  if (!linkedInstallOnly && choices.length === 0 && !original) return null;

  const tracks = installs
    ? instances.map(installTrack).filter((track) => track !== null)
    : [];
  const versionsLine = installedVersionsLine({
    appTargets,
    installedAppVersions,
  });

  return (
    <>
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
            onSent={installs ? () => setOpen(false) : undefined}
            refresh={read}
          />
        </PopoverContent>
      </Popover>
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
