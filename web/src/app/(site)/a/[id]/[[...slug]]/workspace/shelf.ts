"use client";

import { useCallback, useEffect, useState } from "react";
import { fetchShelf, type ShelfImport } from "@/lib/api/query";

/** useShelf keeps the pieces waiting on the shelf, read once the workspace opens. */
export function useShelf(workId: string, enabled: boolean) {
  const [imports, setImports] = useState<ShelfImport[]>([]);

  const reload = useCallback(() => {
    let live = true;
    fetchShelf(workId)
      .then((found) => {
        if (live) setImports(found);
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

  const release = useCallback((pieceId: string) => {
    setImports((current) =>
      current
        .map((held) => ({
          ...held,
          pieces: held.pieces.filter((piece) => piece.id !== pieceId),
        }))
        .filter((held) => held.pieces.length > 0),
    );
  }, []);

  const pieces = imports.flatMap((held) => held.pieces);
  return { imports, pieces, release, reload };
}
