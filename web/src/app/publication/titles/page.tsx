import { PublicationShell } from "@/components/publication/PublicationShell";
import { TitlesAndBadges } from "@/components/publication/TitlesAndBadges";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Titles and badges",
  "Define the Illarin positions, titles and badges that appear on public profiles, and give them to accounts.",
);

export default function TitlesAndBadgesPage() {
  return (
    <PublicationShell
      heading="Titles and badges"
      hint="These appear on a person's public profile. None of them grants any permission on Illarin."
    >
      <TitlesAndBadges />
    </PublicationShell>
  );
}
