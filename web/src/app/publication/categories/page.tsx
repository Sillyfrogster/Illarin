import { PublicationCategories } from "@/components/publication/PublicationCategories";
import { PublicationShell } from "@/components/publication/PublicationShell";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication categories",
  "Name and order the categories Illarin sorts its writing into.",
);

export default function PublicationCategoriesPage() {
  return (
    <PublicationShell
      heading="Categories"
      hint="What a post can be. Renaming one changes the label a reader sees, never what a post references."
    >
      <PublicationCategories />
    </PublicationShell>
  );
}
