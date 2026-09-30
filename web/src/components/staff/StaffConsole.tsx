"use client";

import { ArrowLeft, Search } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";
import { BrandMark } from "@/components/brand/BrandMark";
import { AppearanceMenu } from "@/components/layout/AppearanceMenu";
import { NothingHere } from "@/components/layout/NothingHere";
import { Badge } from "@/components/ui/badge";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";
import { Kbd } from "@/components/ui/kbd";
import {
  Sidebar,
  SidebarBody,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
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
    <SidebarProvider>
      <Workbench account={account}>{children}</Workbench>
    </SidebarProvider>
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
  const { setOpenOnPhone } = useSidebar();
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

  return (
    <>
      <Sidebar label="Staff console">
        <SidebarHeader>
          <Link
            className="flex min-h-control items-center gap-2.5 rounded-control px-1"
            href="/staff"
            onClick={() => setOpenOnPhone(false)}
          >
            <BrandMark size={24} tone="accent" />
            <span className="font-display text-ui font-medium whitespace-nowrap text-ink">
              Staff console
            </span>
          </Link>
        </SidebarHeader>
        <SidebarBody>
          {groupsFor(account.role).map(([group, sections]) => (
            <SidebarGroup key={group} label={group}>
              {sections.map((section) => (
                <SidebarItem
                  badge={
                    section.count ? (
                      <Badge size="compact" tone="accent">
                        {section.count()}
                      </Badge>
                    ) : undefined
                  }
                  current={section === current}
                  href={section.href}
                  icon={<section.icon aria-hidden="true" />}
                  key={section.href}
                  onClick={() => setOpenOnPhone(false)}
                >
                  {section.label}
                </SidebarItem>
              ))}
            </SidebarGroup>
          ))}
        </SidebarBody>
        <SidebarFooter>
          <p className="min-w-0 font-ui text-meta">
            <span className="block truncate font-medium text-ink">
              @{account.handle}
            </span>
            <span className="block text-mute">{ROLE_NAMES[account.role]}</span>
          </p>
          <p className="mt-3 border-t border-rule pt-3 font-ui text-label text-mute">
            Totals are written each night at 00:20 UTC.
          </p>
          <Link
            className="mt-1 inline-flex min-h-control items-center gap-2 font-ui text-meta text-mute hover:text-ink"
            href="/"
          >
            <ArrowLeft aria-hidden="true" className="size-4 shrink-0" />
            Back to Illarin
          </Link>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="sticky top-0 z-20 flex h-13 shrink-0 items-center gap-2 border-b border-rule bg-field/85 px-2 backdrop-blur-sm sm:px-4">
          <SidebarTrigger />
          <Breadcrumb className="min-w-0">
            <BreadcrumbList>
              <BreadcrumbItem className="hidden sm:inline-flex">
                <BreadcrumbLink asChild>
                  <Link href="/staff">Staff</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
              <BreadcrumbSeparator className="hidden sm:block" />
              <BreadcrumbItem>
                <BreadcrumbPage>{current?.label ?? "Console"}</BreadcrumbPage>
              </BreadcrumbItem>
            </BreadcrumbList>
          </Breadcrumb>
          <div className="ml-auto flex items-center gap-1">
            <Button
              className="gap-2 text-mute sm:pr-2"
              onClick={() => setSearching(true)}
              size="compact"
              variant="ghost"
            >
              <Search aria-hidden="true" />
              <span className="sr-only sm:not-sr-only">Search</span>
              <Kbd className="hidden sm:inline-flex">Ctrl K</Kbd>
            </Button>
            <AppearanceMenu />
          </div>
        </header>
        <div className="min-w-0 flex-1 px-3 py-4 sm:px-4 sm:py-5 2xl:px-8">
          <h1 className="sr-only">{current?.label ?? "Console"}</h1>
          {children}
        </div>
      </SidebarInset>

      <ConsolePalette
        onOpenChange={setSearching}
        open={searching}
        sections={sectionsFor(account.role)}
      />
    </>
  );
}
