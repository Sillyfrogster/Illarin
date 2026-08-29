import { ContributorWorkspace } from "@/components/publication/ContributorWorkspace";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Your posts",
  "Everything you have written for the Illarin blog.",
);

export default function PublicationWorkspacePage() {
  return <ContributorWorkspace />;
}
