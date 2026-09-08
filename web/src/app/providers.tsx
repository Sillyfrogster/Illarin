"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { MotionConfig } from "framer-motion";
import { type ReactNode, useState } from "react";
import { makeQueryClient } from "@/lib/api/query";
import { AuthProvider } from "@/lib/auth";

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(makeQueryClient);
  return (
    <QueryClientProvider client={client}>
      <MotionConfig reducedMotion="user">
        <AuthProvider>{children}</AuthProvider>
      </MotionConfig>
    </QueryClientProvider>
  );
}
