import Image from "next/image";
import { Shell } from "@/components/layout/Shell";
import { pageMetadata } from "@/lib/site-metadata";
import { UploadFlow } from "./UploadFlow";

export const metadata = pageMetadata(
  "Publish an asset",
  "Import a file you already have, or start a new character, lorebook, preset, theme or pack.",
);

export default function UploadPage() {
  return (
    <Shell className="pt-10 lg:pt-14">
      <div className="grid items-start gap-10 lg:grid-cols-[minmax(0,1fr)_auto] lg:gap-16">
        <div className="min-w-0 max-w-[38rem]">
          <h1 className="font-display text-display font-medium tracking-[-0.04em] text-ink text-balance">
            Bring in your work
          </h1>
          <p className="mt-4 text-lede text-mute">
            Import a file you already have, or start one from nothing. Both open
            a private draft that only you can see until you publish it.
          </p>
          <UploadFlow />
        </div>
        <div className="w-full max-w-[22rem] sm:max-w-[26rem] lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:w-[clamp(21rem,34vw,30rem)] lg:max-w-none">
          <Image
            alt=""
            className="h-auto w-full rounded-plate shadow-cover"
            height={1402}
            priority
            sizes="(max-width: 639px) 22rem, (max-width: 1023px) 26rem, 30rem"
            src="/publish/watcher-studio.png"
            width={1122}
          />
        </div>
      </div>
    </Shell>
  );
}
