"use client";

import { useMutation } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext, useState } from "react";
import {
  stopWatchingAsset,
  type WorkFollow,
  watchAsset,
} from "@/lib/api/notifications";
import { rememberNotNow, saidNotNow } from "@/lib/asset-watch";
import { useAuth } from "@/lib/auth";

export type AssetWatching = {
  watch: WorkFollow;
  kind: string;
  pending: boolean;
  failure: string | null;
  change: (start: boolean) => void;
  notNow: boolean;
  sayNotNow: () => void;
};

type Changed = { from: WorkFollow | undefined; to: WorkFollow };

const WatchContext = createContext<AssetWatching | null>(null);

/** Shares the signed-in reader's watch on one asset with every control on its page. */
export function AssetWatchProvider({
  assetId,
  children,
  initial,
  kind,
}: {
  assetId: string;
  children: ReactNode;
  initial: WorkFollow | undefined;
  kind: string;
}) {
  const { account } = useAuth();
  const [changed, setChanged] = useState<Changed | null>(null);
  const [declined, setDeclined] = useState(false);
  const mutation = useMutation({
    mutationFn: (start: boolean) =>
      start ? watchAsset(assetId) : stopWatchingAsset(assetId),
    onSuccess: (to) => setChanged({ from: initial, to }),
  });
  const watch = changed && changed.from === initial ? changed.to : initial;
  const value: AssetWatching | null =
    watch && account !== null
      ? {
          watch,
          kind,
          pending: mutation.isPending,
          failure: mutation.error ? mutation.error.message : null,
          change: (start) => mutation.mutate(start),
          notNow: declined || saidNotNow(assetId),
          sayNotNow: () => {
            rememberNotNow(assetId);
            setDeclined(true);
          },
        }
      : null;
  return (
    <WatchContext.Provider value={value}>{children}</WatchContext.Provider>
  );
}

/** The reader's watch, or null for a reader who is signed out or owns the asset. */
export function useAssetWatching(): AssetWatching | null {
  return useContext(WatchContext);
}
