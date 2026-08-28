import { AuthorityConsole } from "@/components/console/AuthorityConsole";
import { RecognitionConsole } from "@/components/recognition/RecognitionConsole";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Recognition",
  "Define the Illarin positions, titles and badges that show on a public profile, and give them out.",
);

export default function RecognitionPage() {
  return (
    <AuthorityConsole
      eyebrow="Recognition"
      heading="Recognition"
      hint="The jobs and the earned titles and badges Illarin puts on a public profile. None of them grants permission anywhere on Illarin."
    >
      <RecognitionConsole />
    </AuthorityConsole>
  );
}
