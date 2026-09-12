import { redirect } from "next/navigation";
import { PUBLICATION_DOCS } from "@/lib/docs/collections";

export default function DevelopersPage() {
  redirect(PUBLICATION_DOCS.href);
}
