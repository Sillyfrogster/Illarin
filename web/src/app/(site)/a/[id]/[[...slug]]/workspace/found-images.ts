"use client";

import { useCallback, useEffect, useState } from "react";
import { type FoundImage, fetchFoundImages } from "@/lib/api/query";

/** useFoundImages keeps the pictures waiting for the creator, read once the workspace opens. */
export function useFoundImages(workId: string, enabled: boolean) {
  const [pictures, setPictures] = useState<FoundImage[]>([]);

  const reload = useCallback(() => {
    let live = true;
    fetchFoundImages(workId)
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
