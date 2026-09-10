import { PublicationRegister } from "@/components/publication/register/PublicationRegister";
import { AuthorityPage } from "@/components/register/AuthorityPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "The publication",
  "Who writes for the Illarin blog, what they may publish, and where an announcement goes.",
);

export default function PublicationPage() {
  return (
    <AuthorityPage
      heading="The publication"
      hint="Who writes for the Illarin blog, what they may publish, and where an announcement goes."
    >
      <PublicationRegister />
    </AuthorityPage>
  );
}
