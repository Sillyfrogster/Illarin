"use client";

import { useMutation } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext, useState } from "react";
import {
  followWork,
  stopFollowingWork,
  type WorkFollow,
} from "@/lib/api/notifications";
import { useAuth } from "@/lib/auth";
import { rememberNotNow, saidNotNow } from "@/lib/work-follow";

export type WorkFollowing = {
  follow: WorkFollow;
  typeName: string;
  pending: boolean;
  failure: string | null;
  change: (start: boolean) => void;
  notNow: boolean;
  sayNotNow: () => void;
};

type Changed = { from: WorkFollow | undefined; to: WorkFollow };

const FollowContext = createContext<WorkFollowing | null>(null);

/** Shares the signed-in reader's follow on one work with every control on its page. */
export function WorkFollowProvider({
  workId,
  children,
  initial,
  typeName,
}: {
  workId: string;
  children: ReactNode;
  initial: WorkFollow | undefined;
  typeName: string;
}) {
  const { account } = useAuth();
  const [changed, setChanged] = useState<Changed | null>(null);
  const [declined, setDeclined] = useState(false);
  const mutation = useMutation({
    mutationFn: (start: boolean) =>
      start ? followWork(workId) : stopFollowingWork(workId),
    onSuccess: (to) => setChanged({ from: initial, to }),
  });
  const follow = changed && changed.from === initial ? changed.to : initial;
  const value: WorkFollowing | null =
    follow && account !== null
      ? {
          follow,
          typeName,
          pending: mutation.isPending,
          failure: mutation.error ? mutation.error.message : null,
          change: (start) => mutation.mutate(start),
          notNow: declined || saidNotNow(workId),
          sayNotNow: () => {
            rememberNotNow(workId);
            setDeclined(true);
          },
        }
      : null;
  return (
    <FollowContext.Provider value={value}>{children}</FollowContext.Provider>
  );
}

/** Whether the reader follows the work, or null for a reader who is signed out or owns it. */
export function useWorkFollowing(): WorkFollowing | null {
  return useContext(FollowContext);
}
