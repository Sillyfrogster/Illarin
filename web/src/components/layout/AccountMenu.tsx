"use client";

import {
  BadgeCheck,
  ChevronDown,
  CircleUserRound,
  LogIn,
  LogOut,
  Mail,
  PenLine,
  Settings,
  UserPlus,
} from "lucide-react";
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
import { AppearanceMenu } from "./AppearanceMenu";
import { accountDestinations, isCurrentPage } from "./destinations";
import { SIGN_OUT_FAILURE, useSignOut } from "./use-sign-out";

const DESTINATION_ICONS = {
  "View profile": CircleUserRound,
  "Account settings": Settings,
  Writers: PenLine,
  "Profile badges": BadgeCheck,
  "Verify email": Mail,
  "Sign in": LogIn,
  "Create account": UserPlus,
};

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
          className="min-w-11 max-w-48 gap-2.5 text-ink data-[state=open]:bg-deep"
        >
          <CircleUserRound aria-hidden="true" />
          <span className="hidden truncate lg:inline">
            {account ? `@${account.handle}` : "Account"}
          </span>
          <ChevronDown
            aria-hidden="true"
            className="hidden !size-3.5 text-mute lg:block"
          />
          <span className="sr-only">Account menu</span>
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent
        align="end"
        className="w-72 max-w-[calc(100vw-2rem)]"
      >
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

        {destinations.map((destination) => {
          const Icon =
            DESTINATION_ICONS[
              destination.label as keyof typeof DESTINATION_ICONS
            ];
          return (
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
                {Icon ? (
                  <Icon aria-hidden="true" className="text-mute" />
                ) : null}
                {destination.label}
              </Link>
            </DropdownMenuItem>
          );
        })}

        <DropdownMenuSeparator />
        <AppearanceMenu embedded />

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
              <LogOut aria-hidden="true" className="text-mute" />
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
