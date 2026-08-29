import { Shell } from "@/components/layout/Shell";
import { PostWriter } from "@/components/publication/editor/PostWriter";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Write a post",
  "The Illarin publication editor.",
);

export default async function PostEditorPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <Shell>
      <PostWriter id={id} />
    </Shell>
  );
}
