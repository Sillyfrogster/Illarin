import { ApiDesk } from "@/components/publication/api/ApiDesk";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication API",
  "API tokens and requests filled in for your approvals.",
);

export default function PublicationApiPage() {
  return <ApiDesk />;
}
