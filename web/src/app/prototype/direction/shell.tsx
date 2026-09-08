"use client";

import { ChevronRight, Menu } from "lucide-react";
import type { ReactNode } from "react";
import { Action, cn, type Direction, Label, Rule } from "./ui";

const NAV = ["Browse", "Publish", "Blog"];

/** A phone keeps every destination, in a control a thumb can reach */
function PhoneMenu({ tone }: { tone: "ink" | "over" }) {
  return (
    <details className="vd:relative vd:ml-auto vd:sm:hidden">
      <summary
        className={cn(
          "vd:inline-flex vd:size-11 vd:cursor-pointer vd:list-none vd:items-center vd:justify-center vd:rounded-control",
          tone === "over" ? "vd:text-over" : "vd:text-ink",
        )}
        aria-label="Menu"
      >
        <Menu className="vd:size-5" />
      </summary>
      <nav className="vd:absolute vd:top-full vd:right-0 vd:z-30 vd:mt-2 vd:w-52 vd:rounded-plate vd:bg-plane vd:p-2 vd:shadow-[inset_0_0_0_1px_var(--v-rule),0_24px_50px_-24px_rgb(0_0_0/0.5)]">
        {[...NAV, "Sign in"].map((item) => (
          <a
            key={item}
            href="#top"
            className="vd:flex vd:min-h-11 vd:items-center vd:rounded-control vd:px-3 vd:text-ui vd:text-ink vd:hover:bg-ink/6"
          >
            {item}
          </a>
        ))}
      </nav>
    </details>
  );
}

/** One header per direction, because the shell is the first thing that says which one this is */
export function SiteHead({
  direction,
  section,
  over = false,
}: {
  direction: Direction;
  section: string;
  over?: boolean;
}) {
  if (direction === "ledger") {
    return (
      <header className="vd:sticky vd:top-0 vd:z-30 vd:bg-field/92 vd:backdrop-blur-md">
        <div className="vd:mx-auto vd:flex vd:max-w-[86rem] vd:items-center vd:gap-5 vd:px-5 vd:py-3.5 vd:md:px-10">
          <span className="vd:font-display vd:text-[1.375rem] vd:font-medium vd:tracking-tight">
            Illarin
          </span>
          <span
            aria-hidden="true"
            className="vd:hidden vd:h-4 vd:w-px vd:bg-rule vd:opacity-70 vd:sm:block"
          />
          <Label className="vd:hidden vd:text-ink vd:sm:block">{section}</Label>
          <nav className="vd:ml-auto vd:hidden vd:items-center vd:sm:flex">
            {NAV.map((item) => (
              <a
                key={item}
                href="#top"
                className="vd:v-hit vd:px-4 vd:py-2 vd:text-meta vd:text-mute vd:shadow-[inset_1px_0_0_var(--v-hair)] vd:hover:text-ink"
              >
                {item}
              </a>
            ))}
          </nav>
          <PhoneMenu tone="ink" />
        </div>
        <Rule />
      </header>
    );
  }

  const floating = direction === "ambient" && over;
  return (
    <header
      className={cn(
        "vd:sticky vd:top-0 vd:z-30",
        floating
          ? "vd:bg-transparent"
          : "vd:bg-field/85 vd:backdrop-blur-xl vd:shadow-[inset_0_-1px_0_var(--v-hair)]",
      )}
    >
      <div className="vd:mx-auto vd:flex vd:max-w-[86rem] vd:items-center vd:gap-8 vd:px-5 vd:py-4 vd:md:px-10">
        <a
          href="#top"
          className={cn(
            "vd:font-display vd:text-[1.5rem] vd:font-medium vd:tracking-tight",
            floating && "vd:text-over",
          )}
        >
          Illarin
        </a>
        <span
          className={cn(
            "vd:hidden vd:text-meta vd:sm:block",
            floating ? "vd:text-over-mute" : "vd:text-mute",
          )}
        >
          {section}
        </span>
        <nav className="vd:ml-auto vd:hidden vd:items-center vd:gap-1 vd:sm:flex">
          {NAV.map((item) => (
            <a
              key={item}
              href="#top"
              className={cn(
                "vd:rounded-control vd:px-3.5 vd:py-2 vd:text-ui",
                floating
                  ? "vd:text-over-mute vd:hover:text-over"
                  : "vd:text-mute vd:hover:bg-ink/6 vd:hover:text-ink",
              )}
            >
              {item}
            </a>
          ))}
          <Action
            variant={direction === "ambient" ? "tinted" : "primary"}
            className="vd:ml-2"
          >
            Sign in
          </Action>
        </nav>
        <PhoneMenu tone={floating ? "over" : "ink"} />
      </div>
    </header>
  );
}

export function SiteFoot() {
  return (
    <footer className="vd:mt-chapter vd:px-5 vd:pb-28 vd:md:px-10">
      <div className="vd:mx-auto vd:max-w-[86rem]">
        <Rule />
        <div className="vd:flex vd:flex-wrap vd:items-baseline vd:gap-x-8 vd:gap-y-4 vd:pt-6">
          <span className="vd:font-display vd:text-[1.25rem] vd:font-medium">
            Illarin
          </span>
          <p className="vd:max-w-[30ch] vd:text-meta vd:text-mute">
            A cross-application catalog for AI roleplay work.
          </p>
          <nav className="vd:ml-auto vd:flex vd:flex-wrap vd:gap-x-6 vd:gap-y-2 vd:text-meta vd:text-mute">
            {["Terms", "Privacy", "Acceptable use", "DMCA"].map((item) => (
              <a key={item} href="#top" className="vd:hover:text-ink">
                {item}
              </a>
            ))}
          </nav>
        </div>
      </div>
    </footer>
  );
}

export function More({
  children,
  className,
  tone = "ink",
}: {
  children: ReactNode;
  className?: string;
  tone?: "ink" | "over";
}) {
  return (
    <a
      href="#top"
      className={cn(
        "vd:group vd:inline-flex vd:items-center vd:gap-1.5 vd:text-ui vd:font-semibold",
        tone === "over" ? "vd:text-over" : "vd:text-ink",
        className,
      )}
    >
      {children}
      <ChevronRight className="vd:size-4 vd:transition vd:group-hover:translate-x-0.5" />
    </a>
  );
}
