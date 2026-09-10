import { RecognitionRegister } from "@/components/recognition/RecognitionRegister";
import { AuthorityPage } from "@/components/register/AuthorityPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Profile badges",
  "The jobs, titles and badges you put on someone's profile.",
);

export default function RecognitionPage() {
  return (
    <AuthorityPage
      heading="Profile badges"
      hint="The jobs, titles and badges you put on someone's profile. None of them lets anyone do anything on Illarin."
    >
      <RecognitionRegister />
    </AuthorityPage>
  );
}
