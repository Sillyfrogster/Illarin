"use client";

import { motion, useMotionValueEvent, useScroll } from "framer-motion";
import { Plus } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { timing } from "@/lib/timing";
import { AccountMenu } from "./AccountMenu";
import { BrowseMenu } from "./BrowseMenu";
import { PUBLISH } from "./destinations";
import { shellClasses } from "./Shell";

const ROW =
  "flex h-[var(--header-height)] items-center gap-x-2 sm:gap-x-3 md:gap-x-5";

export function SiteHeader() {
  const pathname = usePathname();
  const { scrollY } = useScroll();
  const header = useRef<HTMLElement>(null);
  const scrollDirection = useRef({ direction: 0, anchor: 0 });
  const [hidden, setHidden] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useMotionValueEvent(scrollY, "change", (current) => {
    const previous = scrollY.getPrevious() ?? 0;
    const direction = Math.sign(current - previous);
    setScrolled(current > 0);
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

  useEffect(() => setScrolled(scrollY.get() > 0), [scrollY]);

  return (
    <motion.header
      animate={{ y: hidden ? -160 : 0 }}
      className="sticky top-0 z-80 border-b border-transparent transition-colors duration-240 data-[scrolled=true]:border-rule data-[scrolled=true]:bg-field"
      data-scrolled={scrolled}
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
          <BrandLogo className="w-8 sm:hidden" shape="mark" />
          <BrandLogo className="w-28 max-sm:hidden" />
        </Link>
        <BrowseMenu />
        <div className="ml-auto flex items-center sm:gap-1">
          <Button
            asChild
            className="publish max-sm:px-3 sm:mr-1 max-sm:[&_svg]:hidden"
            variant="primary"
          >
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
