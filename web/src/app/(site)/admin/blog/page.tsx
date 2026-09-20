import { BlogAdmin } from "@/components/blog/admin/BlogAdmin";
import { AdminPage } from "@/components/register/AdminPage";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Blog administration",
  "Manage blog writers, categories and announcements.",
);

export default function BlogAdminPage() {
  return (
    <AdminPage
      heading="Blog administration"
      hint="Manage blog writers, categories and announcements."
    >
      <BlogAdmin />
    </AdminPage>
  );
}
