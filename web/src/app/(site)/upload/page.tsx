import { LockKeyhole } from "lucide-react";
import { Shell } from "@/components/layout/Shell";
import { fetchBuildChoices } from "@/lib/api/query";
import { pageMetadata } from "@/lib/site-metadata";
import { UploadFlow } from "./UploadFlow";

export const dynamic = "force-dynamic";

export const metadata = pageMetadata(
  "Upload",
  "Upload a file you already have, or start a new character, lorebook, preset, theme or pack.",
);

export default async function UploadPage() {
  const choices = await fetchBuildChoices().catch(() => null);

  return (
    <Shell className="pt-12 pb-20 lg:pt-14">
      <h1 className="font-display text-[clamp(2rem,3vw,2.75rem)] leading-[1.1] font-medium tracking-[-0.035em] text-ink text-balance">
        Publish your work
      </h1>
      <p className="mt-3 flex items-center gap-2 text-ui text-mute">
        <LockKeyhole
          aria-hidden="true"
          className="size-4 shrink-0 text-accent"
        />
        Drafts stay private until you publish them.
      </p>
      <UploadFlow choices={choices} />
    </Shell>
  );
}
