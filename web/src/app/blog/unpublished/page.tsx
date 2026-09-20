import type { Metadata } from "next";
import { headers } from "next/headers";
import { notFound } from "next/navigation";
import { Unpublished } from "@/components/blog/Unpublished";
import {
  fetchUnpublishedPost,
  UNPUBLISHED_HEADER,
} from "@/lib/post-unpublishing";

export const metadata: Metadata = {
  title: "Unpublished",
  robots: { index: false, follow: false },
};

export default async function UnpublishedPage() {
  const asked = (await headers()).get(UNPUBLISHED_HEADER);
  if (!asked) notFound();
  const unpublished = await fetchUnpublishedPost(asked);
  if (!unpublished) notFound();
  return <Unpublished explanation={unpublished.explanation} />;
}
