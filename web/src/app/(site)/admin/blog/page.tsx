import { BlogAdmin } from "@/components/blog/admin/BlogAdmin";
import { AuthorityPage } from "@/components/register/AuthorityPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Blog administration",
  "Manage blog writers, categories and announcements.",
);

export default function BlogAdminPage() {
  return (
    <AuthorityPage
      heading="Blog administration"
      hint="Manage blog writers, categories and announcements."
    >
      <BlogAdmin />
    </AuthorityPage>
  );
}
