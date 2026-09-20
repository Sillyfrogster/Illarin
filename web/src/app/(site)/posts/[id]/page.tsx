import { PostWriter } from "@/components/blog/writing/PostWriter";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Write a post",
  "The Illarin blog editor.",
);

export default async function PostEditorPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <PostWriter id={id} />;
}
