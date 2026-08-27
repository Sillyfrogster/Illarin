"use client";

import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { components } from "@/lib/api/schema";

export type SignedInAccount = components["schemas"]["Account"];

type AuthContextValue = {
  account: SignedInAccount | null | undefined;
  publicationAuthority: boolean;
  refresh: () => Promise<void>;
  setAccount: (account: SignedInAccount | null) => void;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<SignedInAccount | null | undefined>(
    undefined,
  );
  const [publicationAuthority, setPublicationAuthority] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const response = await fetch("/api/v1/auth/session", {
        cache: "no-store",
        credentials: "same-origin",
      });
      if (!response.ok) {
        setAccount(null);
        setPublicationAuthority(false);
        return;
      }
      const state = (await response.json()) as {
        user: SignedInAccount | null;
        publicationAuthority: boolean;
      };
      setAccount(state.user);
      setPublicationAuthority(state.publicationAuthority);
    } catch {
      setAccount(null);
      setPublicationAuthority(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const signOut = useCallback(async () => {
    const response = await browserFetch("/api/v1/auth/sign-out", {
      method: "POST",
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error("Could not sign out");
    setAccount(null);
    setPublicationAuthority(false);
  }, []);

  const value = useMemo(
    () => ({ account, publicationAuthority, refresh, setAccount, signOut }),
    [account, publicationAuthority, refresh, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
