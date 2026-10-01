"use client";

import { CircleUserRound, LogOut } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
import { cn } from "@/lib/cn";
import { portraitGround } from "@/lib/portrait-tone";
import { AppearanceChoices } from "./AppearanceChoices";
import { DestinationIcon } from "./DestinationIcon";
import { accountDestinations, isCurrentPage } from "./destinations";
import { SIGN_OUT_FAILURE, useSignOut } from "./use-sign-out";

/** AccountMenu is the header's avatar: your pages, the theme and sign out, or sign in and create an account. */
export function AccountMenu() {
  const pathname = usePathname();
  const { account, avatar, writer } = useAuth();
  const [open, setOpen] = useState(false);
  const { signingOut, failed, signOut } = useSignOut(() => setOpen(false));

  if (account === undefined) return <span className="block size-control" />;
  const destinations = accountDestinations(account, writer);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button className="rounded-full text-ink" size="icon" variant="ghost">
          {account ? (
            <Portrait
              className="size-7"
              handle={account.handle}
              picture={avatar}
            />
          ) : (
            <CircleUserRound aria-hidden="true" />
          )}
          <span className="sr-only">Account menu</span>
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-72">
        {account ? (
          <DropdownMenuLabel className="flex items-center gap-3 pt-2">
            <Portrait
              className="size-10 text-lede"
              handle={account.handle}
              picture={avatar}
            />
            <span className="min-w-0">
              <span className="block truncate text-ui font-medium text-ink">
                @{account.handle}
              </span>
              <span className="block text-meta text-mute">
                {account.emailVerified
                  ? "Email verified"
                  : "Verify your email to publish"}
              </span>
            </span>
          </DropdownMenuLabel>
        ) : (
          <DropdownMenuLabel>Not signed in</DropdownMenuLabel>
        )}

        {destinations.map((destination) => {
          const current = isCurrentPage(pathname, destination.href);
          return (
            <DropdownMenuItem
              key={destination.href}
              asChild
              data-current={current ? "page" : undefined}
            >
              <Link
                aria-current={current ? "page" : undefined}
                href={destination.href}
              >
                <DestinationIcon id={destination.id} />
                {destination.label}
              </Link>
            </DropdownMenuItem>
          );
        })}

        <DropdownMenuSeparator />
        <AppearanceChoices />

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
              <p role="alert" className="px-2 pt-1 pb-2 text-meta text-stop">
                {SIGN_OUT_FAILURE}
              </p>
            ) : null}
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Portrait({
  className,
  handle,
  picture,
}: {
  className: string;
  handle: string;
  picture: string | undefined;
}) {
  return (
    <Avatar className={cn("rounded-full", className)}>
      {picture ? <AvatarImage alt="" src={picture} /> : null}
      <AvatarFallback
        aria-hidden="true"
        className={portraitGround(handle)}
        delayMs={picture ? 600 : 0}
      >
        {handle.slice(0, 1).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
}
