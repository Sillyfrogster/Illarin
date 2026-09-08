"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth";

export const SIGN_OUT_FAILURE =
  "Could not sign out. Check your connection and try again.";

/** Signing out, with the two states the shell has to show while it happens */
export function useSignOut(onSignedOut: () => void) {
  const { signOut } = useAuth();
  const [signingOut, setSigningOut] = useState(false);
  const [failed, setFailed] = useState(false);

  async function run() {
    setSigningOut(true);
    setFailed(false);
    try {
      await signOut();
      onSignedOut();
    } catch {
      setFailed(true);
    } finally {
      setSigningOut(false);
    }
  }

  return { signingOut, failed, signOut: () => void run() };
}
