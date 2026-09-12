"use client";

import { createContext, type ReactNode, useContext, useMemo } from "react";

/** The two public origins, as the server knows them. Browser code reads them from here rather than from settings it cannot see. */
export type Origins = { site: string; blog: string };

const OriginsContext = createContext<Origins | null>(null);

export function OriginsProvider({
  site,
  blog,
  children,
}: Origins & { children: ReactNode }) {
  const origins = useMemo(() => ({ site, blog }), [site, blog]);
  return (
    <OriginsContext.Provider value={origins}>
      {children}
    </OriginsContext.Provider>
  );
}

export function useOrigins(): Origins {
  const origins = useContext(OriginsContext);
  if (!origins) throw new Error("useOrigins needs an OriginsProvider above it");
  return origins;
}

/** The blog's hostname as a reader would type it, for showing where a post will live. */
export function useBlogHost(): string {
  return new URL(useOrigins().blog).host;
}
