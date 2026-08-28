import { PublicationApps } from "@/components/publication/PublicationApps";
import { PublicationShell } from "@/components/publication/PublicationShell";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication apps",
  "Configure the projects Illarin publishes official updates for.",
);

export default function PublicationAppsPage() {
  return (
    <PublicationShell
      heading="Apps"
      hint="The projects Illarin publishes for. Configuring one approves nobody."
    >
      <PublicationApps />
    </PublicationShell>
  );
}
