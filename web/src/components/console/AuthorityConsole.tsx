"use client";

import type { ReactNode } from "react";
import { Gate } from "@/components/ui/gate";
import { useAuth } from "@/lib/auth";
import { ConsolePage } from "./ConsolePage";

/** An administration page behind the one account that runs Illarin's publication. */
export function AuthorityConsole({
  children,
  heading,
  hint,
}: {
  children: ReactNode;
  heading: string;
  hint: string;
}) {
  const { account, publicationAuthority } = useAuth();

  return (
    <ConsolePage heading={heading} hint={hint}>
      {account === undefined ? (
        <p aria-live="polite" className="font-ui text-ui text-mute">
          Checking your account…
        </p>
      ) : null}
      {account !== undefined && (!account || !publicationAuthority) ? (
        <Gate
          action={account ? "Back to Illarin" : "Sign in"}
          heading="One recorded account manages this"
          href={account ? "/" : "/sign-in"}
          line="Being an admin or a moderator does not carry it."
        />
      ) : null}
      {account && publicationAuthority ? children : null}
    </ConsolePage>
  );
}
