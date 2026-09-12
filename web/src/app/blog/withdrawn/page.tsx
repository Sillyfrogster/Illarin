import type { Metadata } from "next";
import { headers } from "next/headers";
import { notFound } from "next/navigation";
import { Withdrawn } from "@/components/publication/Withdrawn";
import {
  fetchWithdrawnPost,
  WITHDRAWN_HEADER,
} from "@/lib/publication-withdrawal";

export const metadata: Metadata = {
  title: "Withdrawn",
  robots: { index: false, follow: false },
};

export default async function WithdrawnPage() {
  const asked = (await headers()).get(WITHDRAWN_HEADER);
  if (!asked) notFound();
  const withdrawn = await fetchWithdrawnPost(asked);
  if (!withdrawn) notFound();
  return <Withdrawn explanation={withdrawn.explanation} />;
}
