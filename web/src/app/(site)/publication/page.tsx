import { AuthorityConsole } from "@/components/console/AuthorityConsole";
import { PublicationHub } from "@/components/publication/PublicationHub";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Writers",
  "Who may write for the Illarin blog, for which project, and what kind of post they may write.",
);

export default function PublicationPage() {
  return (
    <AuthorityConsole
      eyebrow="Writers"
      heading="Writers"
      hint="Who may write for the Illarin blog, for which project, and what kind of post they may write."
    >
      <PublicationHub />
    </AuthorityConsole>
  );
}
