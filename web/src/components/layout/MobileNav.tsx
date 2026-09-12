"use client";

import { CircleUserRound, Menu, X } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useAuth } from "@/lib/auth";
import { useOrigins } from "@/lib/origins";
import { AppearanceMenu } from "./AppearanceMenu";
import { DestinationIcon } from "./DestinationIcon";
import {
  accountDestinations,
  isCurrentPage,
  primaryDestinations,
  publishAction,
} from "./destinations";
import { SIGN_OUT_FAILURE, useSignOut } from "./use-sign-out";

const ROW =
  "flex min-h-12 items-center rounded-control px-3 text-ui text-ink hover:bg-deep aria-[current=page]:text-accent";

export function MobileNav() {
  const pathname = usePathname();
  const { account, publicationAuthority } = useAuth();
  const { blog } = useOrigins();
  const [open, setOpen] = useState(false);
  const { signingOut, failed, signOut } = useSignOut(() => setOpen(false));
  const publish = publishAction(account);
  const destinations = accountDestinations(account, publicationAuthority);

  const previousPathname = useRef(pathname);
  useEffect(() => {
    if (previousPathname.current === pathname) return;
    previousPathname.current = pathname;
    setOpen(false);
  }, [pathname]);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden">
          <Menu aria-hidden="true" />
          <span className="sr-only">Open navigation</span>
        </Button>
      </SheetTrigger>
      <SheetContent className="md:hidden">
        <div className="flex min-h-14 items-center justify-between px-[calc(var(--gutter)-0.75rem)] shadow-[inset_0_-1px_0_var(--v-rule)]">
          <SheetTitle className="px-3">
            <BrandLogo />
          </SheetTitle>
          <SheetClose asChild>
            <Button variant="ghost" size="icon">
              <X aria-hidden="true" />
              <span className="sr-only">Close navigation</span>
            </Button>
          </SheetClose>
        </div>
        <SheetDescription className="sr-only">
          Site navigation and account
        </SheetDescription>

        <nav className="grid px-[calc(var(--gutter)-0.75rem)] pt-3">
          {primaryDestinations(blog).map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={ROW}
              aria-current={
                isCurrentPage(pathname, item.href) ? "page" : undefined
              }
            >
              {item.label}
            </Link>
          ))}
          <Button asChild variant="primary" className="my-3">
            <Link href={publish.href}>{publish.label}</Link>
          </Button>

          {account ? (
            <p className="flex items-center gap-2.5 px-3 pt-2 pb-3 text-meta text-mute">
              <CircleUserRound aria-hidden="true" className="size-4" />
              <span className="min-w-0 break-words">
                <span className="block text-ui text-ink">
                  @{account.handle}
                </span>
                {account.emailVerified
                  ? "Email verified"
                  : "Email verification needed"}
              </span>
            </p>
          ) : null}

          {destinations.map((destination) => (
            <Link
              key={destination.href}
              href={destination.href}
              className={`${ROW} gap-2.5`}
              aria-current={
                isCurrentPage(pathname, destination.href) ? "page" : undefined
              }
            >
              <DestinationIcon id={destination.id} />
              {destination.label}
            </Link>
          ))}

          <div className="mt-3 flex min-h-12 items-center justify-between gap-3 border-t border-rule/70 pt-3 pl-3 text-ui text-mute">
            Appearance
            <AppearanceMenu />
          </div>

          {account ? (
            <>
              <Button
                variant="secondary"
                loading={signingOut}
                onClick={signOut}
                className="mt-2"
              >
                {signingOut ? "Signing out…" : "Sign out"}
              </Button>
              {failed ? (
                <p role="alert" className="px-3 pt-2 text-meta text-stop">
                  {SIGN_OUT_FAILURE}
                </p>
              ) : null}
            </>
          ) : null}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
