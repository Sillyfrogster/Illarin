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
import { api } from "@/lib/api/client";
import type { Account, SessionState } from "@/lib/api/shapes";

export type SignedInAccount = Account;

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
      const { data: state } = await api<SessionState>(
        "GET",
        "/v1/auth/session",
        { cache: "no-store" },
      );
      if (!state) {
        setAccount(null);
        setPublicationAuthority(false);
        return;
      }
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
    const { response } = await api<void>("POST", "/v1/auth/sign-out");
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
