import Image from "next/image";
import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";

export function AuthPage({
  children,
  introduction,
  title,
}: {
  children: ReactNode;
  introduction: string;
  title: ReactNode;
}) {
  return (
    <Shell className="max-w-[76rem] pt-12 pb-16 lg:pt-16">
      <div className="grid overflow-hidden rounded-plate bg-inset md:grid-cols-[minmax(0,1fr)_minmax(0,0.9fr)]">
        <div className="min-w-0 px-6 py-8 sm:px-10 sm:py-10 lg:px-12 lg:py-12">
          <h1 className="font-display text-[clamp(1.85rem,3vw,2.5rem)] leading-[1.1] font-medium tracking-[-0.035em] text-balance">
            {title}
          </h1>
          <p className="mt-3 max-w-[46ch] font-prose text-ui text-mute">
            {introduction}
          </p>
          <div className="mt-7">{children}</div>
        </div>

        <div className="relative isolate hidden min-w-0 flex-col justify-between overflow-hidden bg-accent-wash md:flex">
          <div className="relative z-1 px-8 pt-10 lg:px-10 lg:pt-12">
            <p className="max-w-[14ch] font-display text-[clamp(1.8rem,2.6vw,2.75rem)] leading-[1.15] font-medium tracking-[-0.025em] text-ink text-balance">
              A home for your creations.
            </p>
            <p className="mt-4 max-w-[30ch] text-ui text-ink/75">
              Characters, lorebooks, presets, themes and packs. Together in
              Illarin.
            </p>
          </div>
          <Image
            alt=""
            className="mt-8 h-auto w-full object-contain select-none"
            height={1254}
            priority
            sizes="(max-width: 767px) 1px, (max-width: 1279px) 42vw, 470px"
            src="/landing/watcher-portrait.webp"
            width={1254}
          />
        </div>
      </div>
    </Shell>
  );
}
