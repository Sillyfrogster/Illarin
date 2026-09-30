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
import { applyArtwork, readArtwork } from "@/lib/artwork";

export type SignedInAccount = Account;

type AuthContextValue = {
  account: SignedInAccount | null | undefined;
  writer: boolean;
  artwork: boolean;
  setArtwork: (on: boolean) => Promise<void>;
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
  const [artwork, setArtworkState] = useState(true);

  useEffect(() => setArtworkState(readArtwork()), []);

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
      setArtworkState(state.artwork);
      applyArtwork(state.artwork);
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

  const setArtwork = useCallback(
    async (on: boolean) => {
      setArtworkState(on);
      applyArtwork(on);
      if (!account) return;
      const { response } = await api<void>("PUT", "/v1/account/artwork", {
        body: { on },
      });
      if (!response.ok) throw new Error("Could not save the artwork setting");
    },
    [account],
  );

  const value = useMemo(
    () => ({
      account,
      artwork,
      setArtwork,
      refresh,
      setAccount,
      signOut,
      writer,
    }),
    [account, artwork, refresh, setArtwork, signOut, writer],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
