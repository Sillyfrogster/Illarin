"use client";

import { motion, useMotionValueEvent, useScroll } from "framer-motion";
import { Plus } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { Button } from "@/components/ui/button";
import { cn, navLink } from "@/lib/cn";
import { timing } from "@/lib/timing";
import { AccountMenu } from "./AccountMenu";
import { BROWSE, isCurrentPage, PUBLISH } from "./destinations";
import { HeaderSearch } from "./HeaderSearch";
import { shellClasses } from "./Shell";

const ROW =
  "flex min-h-[var(--header-height)] flex-wrap items-center gap-x-3 gap-y-2 py-2 md:flex-nowrap sm:gap-x-5 md:gap-x-8 md:py-0";

export function SiteHeader() {
  const pathname = usePathname();
  const { scrollY } = useScroll();
  const header = useRef<HTMLElement>(null);
  const scrollDirection = useRef({ direction: 0, anchor: 0 });
  const [hidden, setHidden] = useState(false);

  useMotionValueEvent(scrollY, "change", (current) => {
    const previous = scrollY.getPrevious() ?? 0;
    const direction = Math.sign(current - previous);
    if (current < 120) {
      setHidden(false);
      scrollDirection.current = { direction, anchor: current };
      return;
    }
    if (direction !== scrollDirection.current.direction) {
      scrollDirection.current = { direction, anchor: previous };
    }
    if (Math.abs(current - scrollDirection.current.anchor) < 12) return;
    if (direction < 0) setHidden(false);
    if (direction > 0 && !header.current?.contains(document.activeElement))
      setHidden(true);
  });

  useEffect(() => {
    if (pathname) setHidden(false);
  }, [pathname]);

  return (
    <motion.header
      animate={{ y: hidden ? -160 : 0 }}
      className="sticky top-0 z-80 border-b border-rule bg-field"
      data-site-header-hidden={hidden}
      initial={false}
      onFocusCapture={() => setHidden(false)}
      ref={header}
      transition={timing.settle}
    >
      <div className={cn(shellClasses, ROW)}>
        <Link
          href="/"
          aria-label="Illarin home"
          className="flex min-h-control items-center text-ink"
        >
          <BrandLogo className="w-20 sm:w-28" />
        </Link>
        <Link
          aria-current={
            isCurrentPage(pathname, BROWSE.href) ? "page" : undefined
          }
          className={navLink}
          href={BROWSE.href}
        >
          {BROWSE.label}
        </Link>
        <div className="flex items-center gap-2 max-md:contents md:ml-auto">
          <Suspense fallback={<div className="h-control md:w-64" />}>
            <HeaderSearch />
          </Suspense>
        </div>
        <div className="flex items-center gap-0.5 max-md:order-2 max-md:ml-auto">
          <Button asChild className="mr-1" variant="primary">
            <Link
              aria-current={pathname === PUBLISH.href ? "page" : undefined}
              href={PUBLISH.href}
            >
              <Plus aria-hidden="true" />
              {PUBLISH.label}
            </Link>
          </Button>
          <NotificationBell />
          <AccountMenu />
        </div>
      </div>
    </motion.header>
  );
}
