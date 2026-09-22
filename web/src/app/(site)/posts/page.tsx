import { Suspense } from "react";
import { PostDesk } from "@/components/blog/writing/PostDesk";
import { Waiting } from "@/components/ui/waiting";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Your posts",
  "Everything you have written for the Illarin blog.",
);

export default function YourPostsPage() {
  return (
    <Suspense fallback={<Waiting>Loading your posts…</Waiting>}>
      <PostDesk />
    </Suspense>
  );
}
