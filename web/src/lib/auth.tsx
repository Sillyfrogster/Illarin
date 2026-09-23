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
  writer: boolean;
  refresh: () => Promise<void>;
  setAccount: (account: SignedInAccount | null) => void;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<SignedInAccount | null | undefined>(
    undefined,
  );
  const [writer, setWriter] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const { data: state } = await api<SessionState>(
        "GET",
        "/v1/auth/session",
        { cache: "no-store" },
      );
      if (!state) {
        setAccount(null);
        setWriter(false);
        return;
      }
      setAccount(state.user);
      setWriter(state.writer);
    } catch {
      setAccount(null);
      setWriter(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const signOut = useCallback(async () => {
    const { response } = await api<void>("POST", "/v1/auth/sign-out");
    if (!response.ok) throw new Error("Could not sign out");
    setAccount(null);
    setWriter(false);
  }, []);

  const value = useMemo(
    () => ({
      account,
      refresh,
      setAccount,
      signOut,
      writer,
    }),
    [account, refresh, signOut, writer],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
