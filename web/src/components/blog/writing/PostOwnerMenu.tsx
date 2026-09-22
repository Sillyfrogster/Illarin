"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { OwnerMenu } from "@/components/ui/owner-menu";
import { deletePost, recoverPost } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { postPath } from "@/lib/blog-paths";
import { PublishStep, RepublishStep, UnpublishStep } from "./PublishSteps";

export function PostOwnerMenu({
  post,
  onChanged,
  onEdit,
  onSaveFirst,
}: {
  post: Post;
  onChanged: (post: Post) => void;
  onEdit?: () => void;
  onSaveFirst?: () => Promise<number>;
}) {
  const router = useRouter();
  const [failure, setFailure] = useState("");
  const [pending, setPending] = useState(false);
  const shared = {
    post,
    onSettled: onChanged,
    onFailure: setFailure,
    onSaveFirst: onSaveFirst ?? (async () => post.version),
  };
  return (
    <div className="relative z-2">
      {post.deletion ? (
        <Button
          loading={pending}
          onClick={async () => {
            setPending(true);
            const answer = await recoverPost(post.id, post.version);
            setPending(false);
            if (answer.value) {
              setFailure("");
              onChanged(answer.value);
            } else
              setFailure(
                answer.error ??
                  "Illarin could not restore this post. Try again.",
              );
          }}
        >
          Restore post
        </Button>
      ) : (
        <OwnerMenu
          key={post.status}
          name={post.title || "Untitled post"}
          noun="post"
          href={post.status === "published" ? postPath(post.slug) : null}
          onEdit={onEdit ?? (() => router.push(`/posts/${post.id}`))}
          onDelete={async () => {
            const version = await shared.onSaveFirst();
            if (version < 1) throw new Error("Could not save post");
            const answer = await deletePost(post.id, version);
            if (!answer.value) throw new Error(answer.error);
            onChanged(answer.value);
          }}
          visibility={
            <>
              {failure ? (
                <p role="alert" className="mb-4 text-stop">
                  {failure}
                </p>
              ) : null}
              {post.status === "published" ? (
                <UnpublishStep {...shared} />
              ) : post.status === "unpublished" ? (
                <RepublishStep {...shared} />
              ) : (
                <PublishStep {...shared} door="now" />
              )}
            </>
          }
        />
      )}
      {post.deletion && failure ? (
        <p role="alert" className="mt-2 text-meta text-stop">
          {failure}
        </p>
      ) : null}
    </div>
  );
}
