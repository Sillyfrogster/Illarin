"use client";

import { ChevronDown, CircleUserRound } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuth } from "@/lib/auth";
import { accountDestinations, isCurrentPage } from "./destinations";
import { SIGN_OUT_FAILURE, useSignOut } from "./use-sign-out";

/** Everything an account reaches from the shell, behind one trigger */
export function AccountMenu() {
  const pathname = usePathname();
  const { account, publicationAuthority } = useAuth();
  const [open, setOpen] = useState(false);
  const { signingOut, failed, signOut } = useSignOut(() => setOpen(false));
  const destinations = accountDestinations(account, publicationAuthority);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="compact"
          className="max-w-48 gap-1.5 text-ink"
        >
          <CircleUserRound aria-hidden="true" />
          <span className="hidden truncate sm:inline">
            {account ? `@${account.handle}` : "Account"}
          </span>
          <ChevronDown
            aria-hidden="true"
            className="hidden !size-3.5 text-mute sm:block"
          />
          <span className="sr-only">Account menu</span>
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end">
        {account ? (
          <DropdownMenuLabel>
            <span className="block text-ui font-medium break-words">
              @{account.handle}
            </span>
            <span className="mt-0.5 block text-meta text-mute">
              {account.emailVerified
                ? "Verified account"
                : "Email verification needed to publish"}
            </span>
          </DropdownMenuLabel>
        ) : (
          <DropdownMenuLabel className="text-meta text-mute">
            You are not signed in
          </DropdownMenuLabel>
        )}

        {destinations.map((destination) => (
          <DropdownMenuItem
            key={destination.href}
            asChild
            data-current={
              isCurrentPage(pathname, destination.href) ? "page" : undefined
            }
          >
            <Link
              href={destination.href}
              aria-current={
                isCurrentPage(pathname, destination.href) ? "page" : undefined
              }
            >
              {destination.label}
            </Link>
          </DropdownMenuItem>
        ))}

        {account ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={signingOut}
              onSelect={(event) => {
                event.preventDefault();
                signOut();
              }}
            >
              {signingOut ? "Signing out…" : "Sign out"}
            </DropdownMenuItem>
            {failed ? (
              <p role="alert" className="px-3 pt-1 pb-2 text-meta text-stop">
                {SIGN_OUT_FAILURE}
              </p>
            ) : null}
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
