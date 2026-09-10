import { AuthorityConsole } from "@/components/console/AuthorityConsole";
import { PublicationRegister } from "@/components/publication/register/PublicationRegister";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "The publication",
  "Who writes for the Illarin blog, what they may publish, and where an announcement goes.",
);

export default function PublicationPage() {
  return (
    <AuthorityConsole
      heading="The publication"
      hint="Who writes for the Illarin blog, what they may publish, and where an announcement goes."
    >
      <PublicationRegister />
    </AuthorityConsole>
  );
}
