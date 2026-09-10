import { AuthorityConsole } from "@/components/console/AuthorityConsole";
import { RecognitionConsole } from "@/components/recognition/RecognitionConsole";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Profile badges",
  "The jobs, titles and badges you put on someone's profile.",
);

export default function RecognitionPage() {
  return (
    <AuthorityConsole
      heading="Profile badges"
      hint="The jobs, titles and badges you put on someone's profile. None of them lets anyone do anything on Illarin."
    >
      <RecognitionConsole />
    </AuthorityConsole>
  );
}
