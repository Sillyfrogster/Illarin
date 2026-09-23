"use client";

import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";
import { Gate } from "@/components/ui/gate";
import { Waiting } from "@/components/ui/waiting";
import { useAuth } from "@/lib/auth";

export function AdminPage({
  children,
  heading,
  hint,
}: {
  children: ReactNode;
  heading: string;
  hint: string;
}) {
  const { account } = useAuth();
  const admin = account?.role === "admin";

  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          {heading}
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">{hint}</p>
      </header>
      <div className="mt-10 min-w-0">
        {account === undefined ? (
          <Waiting>Checking your account…</Waiting>
        ) : null}
        {account !== undefined && (!account || !admin) ? (
          <Gate
            action={account ? "Back to Illarin" : "Sign in"}
            heading="Admins only"
            href={account ? "/" : "/sign-in"}
            line="Only an admin can use these controls."
          />
        ) : null}
        {account && admin ? children : null}
      </div>
    </Shell>
  );
}
