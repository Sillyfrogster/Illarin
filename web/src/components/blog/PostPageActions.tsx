"use client";

import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { readPost } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";

const PostOwnerMenu = dynamic(() =>
  import("./writing/PostOwnerMenu").then((module) => module.PostOwnerMenu),
);

export function PostPageActions({ id }: { id: string }) {
  const { account, writer } = useAuth();
  const [post, setPost] = useState<Post | null>(null);
  const router = useRouter();
  useEffect(() => {
    let active = true;
    setPost(null);
    if (account && (writer || account.role === "admin")) {
      void readPost(id).then((answer) => {
        if (active) setPost(answer.value ?? null);
      });
    }
    return () => {
      active = false;
    };
  }, [account, writer, id]);
  if (!post) return null;
  return (
    <PostOwnerMenu
      post={post}
      onChanged={(next) => {
        setPost(next);
        if (next.deletion) router.push("/posts?deleted=true");
        else router.refresh();
      }}
    />
  );
}
