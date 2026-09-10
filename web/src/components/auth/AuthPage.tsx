import Image from "next/image";
import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";

const DISSOLVE = "linear-gradient(to bottom, #000 74%, transparent 99%)";

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
    <Shell className="pt-10 pb-chapter lg:pt-16">
      <div className="grid items-start gap-10 md:grid-cols-[minmax(0,30rem)_minmax(0,1fr)] md:gap-12 lg:gap-16">
        <div className="min-w-0">
          <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
            {title}
          </h1>
          <p className="mt-4 max-w-[46ch] font-prose text-lede text-mute">
            {introduction}
          </p>
          <div className="mt-9">{children}</div>
        </div>

        <div className="pointer-events-none hidden justify-self-end md:block lg:sticky lg:top-[calc(var(--header-height)+3rem)]">
          <Image
            alt=""
            className="h-[260px] w-auto max-w-none object-contain select-none lg:h-[380px] xl:h-[430px]"
            height={1254}
            priority
            sizes="(max-width: 1023px) 260px, 430px"
            src="/landing/watcher-portrait.webp"
            style={{ maskImage: DISSOLVE, WebkitMaskImage: DISSOLVE }}
            width={1254}
          />
        </div>
      </div>
    </Shell>
  );
}
