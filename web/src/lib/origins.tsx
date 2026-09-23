"use client";

import { createContext, type ReactNode, useContext, useMemo } from "react";
import { BLOG_HOME } from "./blog-paths";

export type Origins = { site: string };

const OriginsContext = createContext<Origins | null>(null);

export function OriginsProvider({
  site,
  children,
}: Origins & { children: ReactNode }) {
  const origins = useMemo(() => ({ site }), [site]);
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

export function useBlogAddress(): string {
  const address = new URL(BLOG_HOME, useOrigins().site);
  return address.host + address.pathname;
}
