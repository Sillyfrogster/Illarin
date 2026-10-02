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
  identity: OwnIdentity;
  artwork: boolean;
  setArtwork: (on: boolean) => Promise<void>;
  refresh: () => Promise<void>;
  setAccount: (account: SignedInAccount | null) => void;
  signOut: () => Promise<void>;
};

/** OwnIdentity is the name and picture the signed-in account sees on itself. */
export type OwnIdentity = Pick<SessionState, "displayName" | "avatarUrl">;

const NO_IDENTITY: OwnIdentity = {};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [account, setAccount] = useState<SignedInAccount | null | undefined>(
    undefined,
  );
  const [writer, setWriter] = useState(false);
  const [identity, setIdentity] = useState<OwnIdentity>(NO_IDENTITY);
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
        setIdentity(NO_IDENTITY);
        return;
      }
      setAccount(state.user);
      setWriter(state.writer);
      setIdentity({
        displayName: state.displayName,
        avatarUrl: state.avatarUrl,
      });
      setArtworkState(state.artwork);
      applyArtwork(state.artwork);
    } catch {
      setAccount(null);
      setWriter(false);
      setIdentity(NO_IDENTITY);
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
    setIdentity(NO_IDENTITY);
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
      identity,
      setArtwork,
      refresh,
      setAccount,
      signOut,
      writer,
    }),
    [account, artwork, identity, refresh, setArtwork, signOut, writer],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
