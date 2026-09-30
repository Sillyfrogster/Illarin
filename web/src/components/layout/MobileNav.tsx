"use client";

import { CircleUserRound, Menu, X } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { Button } from "@/components/ui/button";
import { MobileDrawer } from "@/components/ui/mobile-drawer";
import { ShinyButton } from "@/components/ui/shiny-button";
import { useAuth } from "@/lib/auth";
import { focusRing } from "@/lib/cn";
import { AppearanceMenu } from "./AppearanceMenu";
import { DestinationIcon } from "./DestinationIcon";
import {
  accountDestinations,
  isCurrentPage,
  primaryDestinations,
  publishAction,
} from "./destinations";
import { SIGN_OUT_FAILURE, useSignOut } from "./use-sign-out";

const ROW = `flex min-h-control items-center rounded-control px-3 text-ui text-ink transition-colors duration-80 hover:bg-hover aria-[current=page]:bg-accent-wash aria-[current=page]:text-accent ${focusRing}`;

export function MobileNav() {
  const pathname = usePathname();
  const { account, writer } = useAuth();
  const [open, setOpen] = useState(false);
  const { signingOut, failed, signOut } = useSignOut(() => setOpen(false));
  const publish = publishAction(account);
  const destinations = accountDestinations(account, writer);

  const trigger = useRef<HTMLButtonElement>(null);
  const previousPathname = useRef(pathname);
  useEffect(() => {
    if (previousPathname.current === pathname) return;
    previousPathname.current = pathname;
    setOpen(false);
  }, [pathname]);

  return (
    <>
      <Button
        aria-expanded={open}
        className="md:hidden"
        onClick={() => setOpen(true)}
        ref={trigger}
        size="icon"
        variant="ghost"
      >
        <Menu aria-hidden="true" />
        <span className="sr-only">Open navigation</span>
      </Button>
      <MobileDrawer
        onClose={() => setOpen(false)}
        open={open}
        title="Navigation"
        triggerRef={trigger}
      >
        <div className="flex items-center justify-between px-3 pt-3 pb-2">
          <BrandLogo className="px-2" />
          <Button onClick={() => setOpen(false)} size="icon" variant="ghost">
            <X aria-hidden="true" />
            <span className="sr-only">Close navigation</span>
          </Button>
        </div>

        <nav className="grid gap-0.5 px-3 pt-1 pb-4">
          {primaryDestinations().map((item) => (
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
          <ShinyButton className="my-3" href={publish.href}>
            {publish.label}
          </ShinyButton>

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
      </MobileDrawer>
    </>
  );
}
