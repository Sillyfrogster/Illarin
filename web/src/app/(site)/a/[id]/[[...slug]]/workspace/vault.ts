"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchVault, type VaultPicture } from "@/lib/api/query";

/** useVault keeps the pictures waiting for the creator, read once the workspace opens. */
export function useVault(workId: string, enabled: boolean) {
  const [pictures, setPictures] = useState<VaultPicture[]>([]);

  const reload = useCallback(() => {
    let live = true;
    fetchVault(workId)
      .then((found) => {
        if (live) setPictures(found);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [workId]);

  useEffect(() => {
    if (!enabled) return;
    return reload();
  }, [enabled, reload]);

  const release = useCallback((pictureId: string) => {
    setPictures((current) =>
      current.filter((picture) => picture.id !== pictureId),
    );
  }, []);

  return { pictures, release, reload };
}
