import { ApiDesk } from "@/components/publication/api/ApiDesk";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Publication API",
  "Create API tokens and use examples for your app's publishing permissions.",
);

export default function PublicationApiPage() {
  return <ApiDesk />;
}
