"use client";

import {
  type InfiniteData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useRef } from "react";
import {
  clearNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification,
  type NotificationList,
  notificationKeys,
  readUnreadCount,
  removeNotification,
} from "@/lib/api/notifications";
import { useAuth } from "@/lib/auth";
import { allMarkedRead, markedRead, removed } from "@/lib/notification-inbox";

/** The signed-in account's unread count, which refreshes whenever the tab regains focus. */
export function useUnreadCount(): number {
  const { account } = useAuth();
  const handle = account?.handle ?? "";
  const query = useQuery({
    queryKey: notificationKeys.unread(handle),
    queryFn: ({ signal }) => readUnreadCount(signal),
    enabled: Boolean(account),
    staleTime: 0,
    refetchOnWindowFocus: "always",
  });
  return account ? (query.data ?? 0) : 0;
}

/** Asks for the unread count again each time the reader moves to another page. */
export function useUnreadRefreshOnNavigation() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const pathname = usePathname();
  const previous = useRef(pathname);
  const handle = account?.handle;

  useEffect(() => {
    if (previous.current === pathname) return;
    previous.current = pathname;
    if (!handle) return;
    void queryClient.invalidateQueries({
      queryKey: notificationKeys.unread(handle),
    });
  }, [pathname, handle, queryClient]);
}

/** Marks an entry read on screen at once, tells the server, and settles every count against its answer. */
export function useOpenEntry() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const handle = account?.handle;

  return useCallback(
    (entry: Notification) => {
      if (!handle || entry.readAt) return;
      const at = new Date().toISOString();
      queryClient.setQueryData<number>(
        notificationKeys.unread(handle),
        (count) => (count === undefined ? count : Math.max(0, count - 1)),
      );
      rewriteLists(queryClient, handle, (page) =>
        markedRead(page, entry.id, at),
      );
      const settle = () =>
        queryClient.invalidateQueries({ queryKey: notificationKeys.all });
      void markNotificationRead(entry.id).then(settle, settle);
    },
    [handle, queryClient],
  );
}

/** Marks every entry read on screen at once and settles against the server's answer. */
export function useMarkAllRead() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const handle = account?.handle;

  return useMutation({
    mutationFn: markAllNotificationsRead,
    onMutate: () => {
      if (!handle) return;
      const at = new Date().toISOString();
      queryClient.setQueryData<number>(notificationKeys.unread(handle), 0);
      rewriteLists(queryClient, handle, (page) => allMarkedRead(page, at));
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: notificationKeys.all }),
  });
}

/** Takes an entry off the screen at once, tells the server, and settles every count against its answer. */
export function useRemoveEntry() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const handle = account?.handle;

  return useCallback(
    (entry: Notification) => {
      if (!handle) return;
      if (!entry.readAt) {
        queryClient.setQueryData<number>(
          notificationKeys.unread(handle),
          (count) => (count === undefined ? count : Math.max(0, count - 1)),
        );
      }
      rewriteLists(queryClient, handle, (page) => removed(page, entry.id));
      const settle = () =>
        queryClient.invalidateQueries({ queryKey: notificationKeys.all });
      void removeNotification(entry.id).then(settle, settle);
    },
    [handle, queryClient],
  );
}

/** Empties the inbox on screen at once and settles against the server's answer. */
export function useClearAll() {
  const { account } = useAuth();
  const queryClient = useQueryClient();
  const handle = account?.handle;

  return useMutation({
    mutationFn: clearNotifications,
    onMutate: () => {
      if (!handle) return;
      queryClient.setQueryData<number>(notificationKeys.unread(handle), 0);
      queryClient.setQueryData<InfiniteData<NotificationList>>(
        notificationKeys.inbox(handle),
        (data) => data && { pageParams: [null], pages: [{ items: [] }] },
      );
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: notificationKeys.all }),
  });
}

function rewriteLists(
  queryClient: ReturnType<typeof useQueryClient>,
  handle: string,
  rewrite: (page: NotificationList) => NotificationList,
) {
  queryClient.setQueryData<InfiniteData<NotificationList>>(
    notificationKeys.inbox(handle),
    (data) => data && { ...data, pages: data.pages.map(rewrite) },
  );
}
