import { ContributorWorkspace } from "@/components/publication/ContributorWorkspace";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "What you may publish",
  "The projects and categories Illarin approved you to publish under.",
);

export default function PublicationWorkspacePage() {
  return <ContributorWorkspace />;
}
