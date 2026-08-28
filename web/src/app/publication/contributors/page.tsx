import { PublicationContributors } from "@/components/publication/PublicationContributors";
import { PublicationShell } from "@/components/publication/PublicationShell";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication contributors",
  "Approve one verified account to publish official updates for one app.",
);

export default function PublicationContributorsPage() {
  return (
    <PublicationShell
      heading="Contributors"
      hint="One approval binds one person to one app. Two people from the same project get an approval each."
    >
      <PublicationContributors />
    </PublicationShell>
  );
}
