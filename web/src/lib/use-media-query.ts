import { useCallback, useSyncExternalStore } from "react";

export const PHONE_WIDTH = "(max-width: 767.98px)";

/** Reports whether a media query matches, answering false on the server. */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (changed: () => void) => {
      const list = window.matchMedia(query);
      list.addEventListener("change", changed);
      return () => list.removeEventListener("change", changed);
    },
    [query],
  );

  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia(query).matches,
    () => false,
  );
}
