import { RecognitionRegister } from "@/components/recognition/RecognitionRegister";
import { AuthorityPage } from "@/components/register/AuthorityPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Profile recognition",
  "Manage profile positions, titles and badges.",
);

export default function RecognitionPage() {
  return (
    <AuthorityPage
      heading="Profile recognition"
      hint="Manage profile positions, titles and badges. These do not grant account permissions."
    >
      <RecognitionRegister />
    </AuthorityPage>
  );
}
