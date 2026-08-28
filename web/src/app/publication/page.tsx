import { AuthorityConsole } from "@/components/console/AuthorityConsole";
import { PublicationHub } from "@/components/publication/PublicationHub";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication",
  "Approve who publishes official updates for a project, and keep the apps and categories they publish under.",
);

export default function PublicationPage() {
  return (
    <AuthorityConsole
      eyebrow="Publication"
      heading="Publication"
      hint="Who may publish official updates on Illarin, for which project, and under which categories."
    >
      <PublicationHub />
    </AuthorityConsole>
  );
}
