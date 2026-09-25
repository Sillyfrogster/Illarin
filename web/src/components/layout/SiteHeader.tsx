"use client";

import {
  motion,
  useMotionTemplate,
  useMotionValueEvent,
  useReducedMotion,
  useScroll,
  useTransform,
} from "framer-motion";
import { Plus } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { Button } from "@/components/ui/button";
import { LineLink } from "@/components/ui/line-link";
import { useAuth } from "@/lib/auth";
import { AccountMenu } from "./AccountMenu";
import {
  isCurrentPage,
  primaryDestinations,
  publishAction,
} from "./destinations";
import { MobileNav } from "./MobileNav";
import { Notch } from "./Notch";

export function SiteHeader() {
  const pathname = usePathname();
  const { account } = useAuth();
  const publish = publishAction(account);
  const { scrollY } = useScroll();
  const depth = useTransform(scrollY, [0, 40], [0.16, 0.36], { clamp: true });
  const lift = useMotionTemplate`drop-shadow(0 6px 10px rgb(0 0 0 / ${depth}))`;
  const reducedMotion = useReducedMotion();
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

  const motionProps = {
    animate: { y: hidden ? -120 : 0 },
    initial: false,
    transition: {
      duration: reducedMotion ? 0 : 0.22,
      ease: "easeOut" as const,
    },
    onFocusCapture: () => setHidden(false),
    ref: header,
  };

  if (pathname === "/")
    return (
      <motion.header
        {...motionProps}
        data-site-header-hidden={hidden}
        data-theme="dark"
        style={{ colorScheme: "dark" }}
        className="fixed inset-x-0 top-0 z-80 flex h-22 items-center gap-6 bg-linear-to-b from-[#100e1699] to-transparent px-[clamp(24px,4.2vw,88px)] font-ui text-ink [--v-ink:#fbf8ff] [--v-mute:#cfc2d8] [--v-deep:#26202c] md:h-27 md:gap-14"
      >
        <Link
          href="/"
          aria-label="Illarin home"
          className="flex min-h-11 items-center text-ink"
        >
          <BrandLogo className="w-28 md:w-32" />
        </Link>
        <nav
          aria-label="Primary"
          className="hidden items-center gap-8 text-meta sm:flex"
        >
          {primaryDestinations().map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="flex min-h-11 items-center text-ink"
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1">
          <MobileNav />
          <NotificationBell />
          <AccountMenu />
        </div>
      </motion.header>
    );

  return (
    <motion.header
      {...motionProps}
      data-site-header-hidden={hidden}
      style={{ filter: lift }}
      className="sticky top-0 z-80 h-[var(--header-height)]"
    >
      <Notch
        start={
          <>
            <MobileNav />
            <nav
              className="hidden items-center gap-2 md:flex"
              aria-label="Primary"
            >
              {primaryDestinations().map((item) => (
                <LineLink
                  key={item.href}
                  href={item.href}
                  current={isCurrentPage(pathname, item.href)}
                  className="rounded-control px-3 text-ink hover:bg-deep aria-[current=page]:bg-deep"
                >
                  {item.label}
                </LineLink>
              ))}
            </nav>
          </>
        }
        centre={
          <Link
            href="/"
            aria-label="Illarin home"
            className="flex min-h-11 items-center gap-2 text-ink"
          >
            <BrandLogo className="sm:w-36" />
          </Link>
        }
        end={
          <>
            <Button asChild variant="primary" className="hidden md:inline-flex">
              <Link
                href={publish.href}
                aria-current={pathname === "/upload" ? "page" : undefined}
              >
                <Plus aria-hidden="true" />
                {publish.label}
              </Link>
            </Button>
            <div className="flex items-center gap-1">
              <NotificationBell />
              <AccountMenu />
            </div>
          </>
        }
      />
    </motion.header>
  );
}
