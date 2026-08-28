"use client";

import { useCallback, useEffect, useState } from "react";
import { readTokens } from "@/lib/api/publication";
import type { PublicationToken } from "@/lib/api/query";

/** Reads one grant's tokens, and reads them again after any change to them. */
export function useGrantTokens(
  grantId: string,
  onFailure: (message: string) => void,
) {
  const [tokens, setTokens] = useState<PublicationToken[] | null>(null);

  const reread = useCallback(async () => {
    const answer = await readTokens(grantId);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    setTokens(answer.value.tokens);
  }, [grantId, onFailure]);

  useEffect(() => {
    void reread();
  }, [reread]);

  return {
    tokens,
    live: tokens?.filter((token) => token.active) ?? [],
    spent: tokens?.filter((token) => !token.active) ?? [],
    reread: () => void reread(),
  };
}
