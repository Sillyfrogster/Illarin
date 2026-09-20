"use client";

import { ArrowLeft, Search } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";
import { BrandMark } from "@/components/brand/BrandMark";
import { AppearanceMenu } from "@/components/layout/AppearanceMenu";
import { NothingHere } from "@/components/layout/NothingHere";
import { Button } from "@/components/ui/button";
import {
  Rail,
  RailBadge,
  RailBody,
  RailFoot,
  RailGroup,
  RailHeader,
  RailInset,
  RailItem,
  RailProvider,
  RailToggle,
  useRail,
} from "@/components/ui/rail";
import { PageWaiting } from "@/components/ui/waiting";
import { type SignedInAccount, useAuth } from "@/lib/auth";
import { ConsolePalette } from "./ConsolePalette";
import {
  groupsFor,
  isStaff,
  ROLE_NAMES,
  type StaffRole,
  sectionAt,
  sectionsFor,
} from "./sections";

/** The console's own shell, with sections down the side and the page's own header above its panels. */
export function StaffConsole({ children }: { children: ReactNode }) {
  const { account } = useAuth();

  if (account === undefined) {
    return <PageWaiting>Checking your account…</PageWaiting>;
  }
  if (!isStaff(account)) return <NothingHere />;

  return (
    <RailProvider>
      <Workbench account={account}>{children}</Workbench>
    </RailProvider>
  );
}

function Workbench({
  account,
  children,
}: {
  account: SignedInAccount & { role: StaffRole };
  children: ReactNode;
}) {
  const pathname = usePathname();
  const { wide, onPhone, setOpenOnPhone } = useRail();
  const [searching, setSearching] = useState(false);
  const current = sectionAt(account.role, pathname);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        setSearching((open) => !open);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const showLabels = wide || onPhone;

  return (
    <>
      <Rail label="Staff console">
        <RailHeader className={showLabels ? "" : "justify-center px-0"}>
          <Link
            className="flex min-h-11 items-center gap-2.5 rounded-control px-1 outline-offset-3"
            href="/staff"
            onClick={() => setOpenOnPhone(false)}
          >
            <BrandMark size={24} tone="accent" />
            <span
              className={
                showLabels
                  ? "font-display text-ui font-medium whitespace-nowrap text-ink"
                  : "sr-only"
              }
            >
              Staff console
            </span>
          </Link>
        </RailHeader>
        <RailBody>
          {groupsFor(account.role).map(([group, sections]) => (
            <RailGroup key={group} label={group}>
              {sections.map((section) => (
                <RailItem
                  badge={
                    section.count ? (
                      <RailBadge>{section.count()}</RailBadge>
                    ) : undefined
                  }
                  current={section === current}
                  href={section.href}
                  icon={<section.icon aria-hidden="true" />}
                  key={section.href}
                  onClick={() => setOpenOnPhone(false)}
                >
                  {section.label}
                </RailItem>
              ))}
            </RailGroup>
          ))}
        </RailBody>
        <RailFoot className={showLabels ? "" : "text-center"}>
          <p className="min-w-0 font-ui text-meta">
            <span className="block truncate font-medium text-ink">
              {showLabels
                ? `@${account.handle}`
                : account.handle[0].toUpperCase()}
            </span>
            <span className={showLabels ? "block text-mute" : "sr-only"}>
              {ROLE_NAMES[account.role]}
            </span>
          </p>
          <p
            className={
              showLabels
                ? "mt-3 border-t border-rule pt-3 font-ui text-label text-mute"
                : "sr-only"
            }
          >
            Totals are written each night at 00:20 UTC.
          </p>
          <Link
            className="mt-1 inline-flex min-h-11 items-center gap-2 rounded-control font-ui text-meta text-mute outline-offset-2 hover:text-ink"
            href="/"
          >
            <ArrowLeft aria-hidden="true" className="size-4 shrink-0" />
            <span className={showLabels ? "" : "sr-only"}>Back to Illarin</span>
          </Link>
        </RailFoot>
      </Rail>

      <RailInset>
        <header className="sticky top-0 z-20 flex h-13 shrink-0 items-center gap-2 border-b border-rule bg-field/85 px-2 backdrop-blur-sm sm:px-4">
          <RailToggle />
          <p className="flex min-w-0 items-baseline gap-2 truncate font-ui text-meta">
            <span className="hidden text-mute sm:inline">Staff</span>
            <span aria-hidden="true" className="hidden text-rule sm:inline">
              /
            </span>
            <span className="truncate font-medium tracking-[0.04em] text-ink uppercase">
              {current?.label ?? "Console"}
            </span>
          </p>
          <div className="ml-auto flex items-center gap-1">
            <Button
              className="gap-2 text-mute sm:pr-2"
              onClick={() => setSearching(true)}
              size="compact"
              variant="ghost"
            >
              <Search aria-hidden="true" />
              <span className="sr-only sm:not-sr-only">Search</span>
              <kbd className="hidden rounded-[5px] bg-deep px-1.5 py-0.5 font-ui text-label text-mute sm:inline">
                Ctrl K
              </kbd>
            </Button>
            <AppearanceMenu />
          </div>
        </header>
        <div className="min-w-0 flex-1 px-3 py-4 sm:px-4 sm:py-5 2xl:px-8">
          {children}
        </div>
      </RailInset>

      <ConsolePalette
        onOpenChange={setSearching}
        open={searching}
        sections={sectionsFor(account.role)}
      />
    </>
  );
}
