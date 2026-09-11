"use client";

import { ChevronDown } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { AssetInstance, AssetInstanceList } from "@/lib/api/query";
import { formatChoices } from "@/lib/asset-delivery";
import { useAuth } from "@/lib/auth";
import { AssetChooser, type AssetChooserProps } from "./AssetChooser";

export function GetAsset(props: AssetChooserProps) {
  const {
    assetId,
    kindLabel,
    downloads,
    appTargets,
    original,
    holdsNothing,
    linkedInstallOnly,
  } = props;
  const { account } = useAuth();
  const [instances, setInstances] = useState<AssetInstance[]>([]);
  const [opened, setOpened] = useState(0);

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

  const choices = formatChoices({
    downloads,
    holdsNothing,
    app: appTargets[0]?.id ?? "",
    apps: appTargets,
  });
  const receiving = instances.some((one) => one.canReceive);
  if (linkedInstallOnly && !receiving) return null;
  if (!linkedInstallOnly && choices.length === 0 && !original) return null;

  return (
    <Popover onOpenChange={() => setOpened((count) => count + 1)}>
      <PopoverTrigger asChild>
        <Button className="min-w-52 justify-between" variant="primary">
          Download {kindLabel}
          <ChevronDown aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" aria-label={`Download this ${kindLabel}`}>
        <AssetChooser
          key={opened}
          {...props}
          instances={instances}
          refresh={read}
        />
      </PopoverContent>
    </Popover>
  );
}
