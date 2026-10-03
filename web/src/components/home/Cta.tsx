import { ArrowRight, ArrowUpRight } from "lucide-react";
import Link from "next/link";

/** Lead is the page's one main action: a slim violet pill whose arrow sits in its own disc and slides through it on hover. */
export function Lead({ children, href }: { children: string; href: string }) {
  return (
    <Link
      className="group/lead inline-flex h-12 items-center gap-4 rounded-full bg-action py-1.5 pr-1.5 pl-6 text-ui font-medium text-on-accent transition-colors duration-200 hover:bg-action-hover"
      href={href}
    >
      {children}
      <span className="relative grid size-9 place-items-center overflow-hidden rounded-full bg-on-accent text-action">
        <ArrowRight
          aria-hidden="true"
          className="size-4 transition-transform duration-300 ease-(--ease-wipe) group-hover/lead:translate-x-8"
        />
        <ArrowRight
          aria-hidden="true"
          className="absolute size-4 -translate-x-8 transition-transform duration-300 ease-(--ease-wipe) group-hover/lead:translate-x-0"
        />
      </span>
    </Link>
  );
}

/** Aside is the quieter second way in: text with an underline that draws in from the left. */
export function Aside({ children, href }: { children: string; href: string }) {
  return (
    <Link
      className="group/aside inline-flex h-12 items-center gap-1.5 text-ui font-medium text-over"
      href={href}
    >
      <span className="bg-linear-to-r from-current to-current bg-size-[0%_1px] bg-left-bottom bg-no-repeat pb-0.5 transition-[background-size] duration-300 ease-(--ease-wipe) group-hover/aside:bg-size-[100%_1px]">
        {children}
      </span>
      <ArrowUpRight
        aria-hidden="true"
        className="size-4 transition-transform duration-300 ease-(--ease-wipe) group-hover/aside:translate-x-0.5 group-hover/aside:-translate-y-0.5"
      />
    </Link>
  );
}
