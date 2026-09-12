"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";
import { makeQueryClient } from "@/lib/api/query";
import { AuthProvider } from "@/lib/auth";

/** The account session and its queries, which signed-in pages need and the read-only blog origin must never load. */
export function SiteProviders({ children }: { children: ReactNode }) {
  const [client] = useState(makeQueryClient);
  return (
    <QueryClientProvider client={client}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
}
