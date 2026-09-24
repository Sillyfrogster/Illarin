import { Check, LockKeyhole, PencilLine, Upload } from "lucide-react";
import Image from "next/image";
import { Shell } from "@/components/layout/Shell";
import { fetchBuildChoices } from "@/lib/api/query";
import { pageMetadata } from "@/lib/site-metadata";
import { StartFromNothing } from "./StartFromNothing";
import { UploadFlow } from "./UploadFlow";

export const metadata = pageMetadata(
  "Upload",
  "Import a file you already have, or start a new character, lorebook, preset, theme or pack.",
);

export default async function UploadPage() {
  const choices = await fetchBuildChoices().catch(() => null);

  return (
    <Shell className="max-w-[78rem] pt-12 pb-16 lg:pt-14">
      <div className="grid items-start gap-10 lg:grid-cols-[minmax(0,1fr)_19rem] lg:gap-16">
        <div className="min-w-0">
          <h1 className="font-display text-[clamp(2rem,3vw,2.75rem)] leading-[1.1] font-medium tracking-[-0.035em] text-ink text-balance">
            Share your work
          </h1>
          <p className="mt-3 max-w-[48ch] text-ui text-mute">
            Import a file you already have, or start a new draft in the editor.
          </p>
          <UploadFlow />
        </div>
        <aside className="grid min-w-0 gap-6 rounded-plate bg-inset p-5 sm:grid-cols-[12rem_1fr] sm:items-center lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:grid-cols-1">
          <Image
            alt=""
            className="mx-auto hidden h-auto w-full max-w-[15rem] rounded-control sm:block sm:max-w-none"
            height={1402}
            sizes="(max-width: 639px) 240px, (max-width: 1023px) 192px, 264px"
            src="/publish/watcher-studio.png"
            width={1122}
          />
          <div>
            <p className="flex items-center gap-2 font-ui text-ui font-medium text-ink">
              <LockKeyhole aria-hidden="true" className="size-4 text-accent" />
              Private until you publish
            </p>
            <ol className="mt-5 grid list-none gap-5 p-0">
              <li className="flex items-start gap-3">
                <Upload
                  aria-hidden="true"
                  className="mt-1 size-4 shrink-0 text-mute"
                />
                <p className="text-meta text-mute">
                  <strong className="block font-medium text-ink">
                    Start a draft
                  </strong>
                  Bring a file or choose a type.
                </p>
              </li>
              <li className="flex items-start gap-3">
                <PencilLine
                  aria-hidden="true"
                  className="mt-1 size-4 shrink-0 text-mute"
                />
                <p className="text-meta text-mute">
                  <strong className="block font-medium text-ink">
                    Make it yours
                  </strong>
                  Edit the details, add media and preview your page.
                </p>
              </li>
              <li className="flex items-start gap-3">
                <Check
                  aria-hidden="true"
                  className="mt-1 size-4 shrink-0 text-mute"
                />
                <p className="text-meta text-mute">
                  <strong className="block font-medium text-ink">
                    Publish when ready
                  </strong>
                  You decide when the page becomes public.
                </p>
              </li>
            </ol>
          </div>
        </aside>
      </div>
      <StartFromNothing choices={choices} />
    </Shell>
  );
}
