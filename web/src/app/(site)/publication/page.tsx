import { PublicationRegister } from "@/components/publication/register/PublicationRegister";
import { AuthorityPage } from "@/components/register/AuthorityPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Blog administration",
  "Manage blog contributors, apps, categories and announcements.",
);

export default function PublicationPage() {
  return (
    <AuthorityPage
      heading="Blog administration"
      hint="Manage blog contributors, apps, categories and announcements."
    >
      <PublicationRegister />
    </AuthorityPage>
  );
}
