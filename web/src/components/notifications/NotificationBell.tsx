"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { Bell, CheckCheck, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  type NotificationCursor,
  notificationKeys,
  readNotifications,
} from "@/lib/api/notifications";
import { useAuth } from "@/lib/auth";
import { unreadBadge, unreadLabel } from "@/lib/notification-inbox";
import {
  NotificationEntry,
  NotificationEntrySkeleton,
} from "./NotificationEntry";
import {
  useClearAll,
  useMarkAllRead,
  useOpenEntry,
  useRemoveEntry,
  useUnreadCount,
  useUnreadRefreshOnNavigation,
} from "./use-inbox";

const PAGE_SIZE = 10;

export function NotificationBell() {
  const { account } = useAuth();
  const [open, setOpen] = useState(false);
  const count = useUnreadCount();
  useUnreadRefreshOnNavigation();
  if (!account) return null;
  const badge = unreadBadge(count);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="text-ink data-[state=open]:bg-deep"
        >
          <Bell aria-hidden="true" />
          <AnimatePresence initial={false}>
            {badge ? (
              <motion.span
                key="badge"
                aria-hidden="true"
                initial={{ scale: 0.5, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                exit={{ scale: 0.5, opacity: 0 }}
                transition={{ type: "spring", stiffness: 560, damping: 30 }}
                className="absolute top-1.5 right-1 grid h-[1.125rem] min-w-[1.125rem] place-items-center rounded-full bg-action px-1 text-label leading-none font-semibold text-on-accent tabular-nums ring-2 ring-plane"
              >
                {badge}
              </motion.span>
            ) : null}
          </AnimatePresence>
          <span className="sr-only">{unreadLabel(count)}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        aria-label="Notifications"
        className="flex w-[min(25rem,calc(100vw-2rem))] flex-col overflow-hidden p-0"
      >
        <NotificationPanel count={count} onLeave={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
}

/** Holds the whole inbox, the latest ten first and older entries a page at a time below them. */
function NotificationPanel({
  count,
  onLeave,
}: {
  count: number;
  onLeave: () => void;
}) {
  const { account } = useAuth();
  const inbox = useInfiniteQuery({
    queryKey: notificationKeys.inbox(account?.handle ?? ""),
    queryFn: ({ pageParam, signal }) =>
      readNotifications(PAGE_SIZE, pageParam, signal),
    initialPageParam: null as NotificationCursor | null,
    getNextPageParam: (last) => last.nextCursor ?? undefined,
    enabled: Boolean(account),
    staleTime: 0,
  });
  const entries = useMemo(
    () => inbox.data?.pages.flatMap((page) => page.items) ?? [],
    [inbox.data],
  );
  const openEntry = useOpenEntry();
  const removeEntry = useRemoveEntry();
  const markAll = useMarkAllRead();
  const clearAll = useClearAll();
  const now = new Date();

  return (
    <>
      <div className="flex min-h-14 shrink-0 items-center justify-between gap-2 py-1.5 pr-2 pl-5">
        <h2 className="flex min-w-0 flex-wrap items-baseline gap-x-2 font-ui text-ui font-medium text-ink">
          Notifications
          {count > 0 ? (
            <span className="text-meta font-normal text-accent tabular-nums">
              {count} unread
            </span>
          ) : null}
        </h2>
        <div className="flex shrink-0 items-center">
          <Button
            variant="ghost"
            size="compact"
            disabled={count === 0 || markAll.isPending}
            onClick={() => markAll.mutate()}
          >
            <CheckCheck aria-hidden="true" />
            Mark all read
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Clear all"
            disabled={entries.length === 0 || clearAll.isPending}
            onClick={() => clearAll.mutate()}
          >
            <Trash2 aria-hidden="true" />
            <span className="sr-only">Clear all notifications</span>
          </Button>
        </div>
      </div>
      {markAll.isError ? (
        <p role="alert" className="px-5 pb-2 text-meta text-stop">
          Your notifications could not be marked read. Try again.
        </p>
      ) : null}
      {clearAll.isError ? (
        <p role="alert" className="px-5 pb-2 text-meta text-stop">
          Your notifications could not be cleared. Try again.
        </p>
      ) : null}

      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-2 pb-2">
        {inbox.data ? (
          entries.length > 0 ? (
            <>
              <ul className="flex list-none flex-col gap-1">
                {entries.map((entry) => (
                  <NotificationEntry
                    key={entry.id}
                    entry={entry}
                    now={now}
                    onOpen={(opened) => {
                      openEntry(opened);
                      onLeave();
                    }}
                    onRemove={removeEntry}
                  />
                ))}
              </ul>
              {inbox.hasNextPage ? (
                <Button
                  variant="ghost"
                  className="mt-1 w-full"
                  loading={inbox.isFetchingNextPage}
                  onClick={() => void inbox.fetchNextPage()}
                >
                  Show older notifications
                </Button>
              ) : (
                <p className="px-3 pt-4 pb-3 text-center text-label text-mute">
                  Notifications are kept for 90 days.
                </p>
              )}
              {inbox.isFetchNextPageError ? (
                <p
                  role="alert"
                  className="px-3 pb-2 text-center text-meta text-stop"
                >
                  Older notifications could not be loaded. Try again.
                </p>
              ) : null}
            </>
          ) : (
            <div className="px-6 pt-5 pb-8 text-center">
              <p className="text-ui text-ink">No notifications</p>
              <p className="mx-auto mt-1 max-w-[30ch] text-meta text-mute">
                Updates to work you follow, and anything Illarin staff do with
                your work, arrive here.
              </p>
            </div>
          )
        ) : inbox.isError ? (
          <div role="alert" className="px-3 pt-4 pb-6 text-center">
            <p className="text-ui text-ink">
              Your notifications could not be loaded.
            </p>
            <Button
              variant="ghost"
              className="mt-2"
              onClick={() => void inbox.refetch()}
            >
              Try again
            </Button>
          </div>
        ) : (
          <ul
            aria-busy="true"
            aria-label="Loading notifications"
            className="list-none"
          >
            <NotificationEntrySkeleton />
            <NotificationEntrySkeleton />
            <NotificationEntrySkeleton />
          </ul>
        )}
      </div>
    </>
  );
}
