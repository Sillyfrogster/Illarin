import {
  LockKeyhole,
  LockKeyholeOpen,
  type LucideIcon,
  ShieldCheck,
  ShieldOff,
} from "lucide-react";
import Link from "next/link";
import type { Notification } from "@/lib/api/notifications";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { arrivedAgo, notificationWords } from "@/lib/notification-inbox";

const TAKEN = "bg-stop-wash text-stop";
const GIVEN_BACK = "bg-accent-wash text-accent";

const MARKS = {
  asset_withheld: { icon: LockKeyhole, tone: TAKEN },
  asset_restored: { icon: LockKeyholeOpen, tone: GIVEN_BACK },
  profile_restricted: { icon: ShieldOff, tone: TAKEN },
  profile_restored: { icon: ShieldCheck, tone: GIVEN_BACK },
} satisfies Record<Notification["type"], { icon: LucideIcon; tone: string }>;

/** One inbox entry, tinted while unread, that opens what it is about when it has somewhere to go. */
export function NotificationEntry({
  entry,
  now,
  onOpen,
}: {
  entry: Notification;
  now: Date;
  onOpen: (entry: Notification) => void;
}) {
  const words = notificationWords(entry);
  const unread = !entry.readAt;
  const mark = MARKS[entry.type];
  const Icon = mark.icon;
  const row = cn(
    "group relative flex gap-3.5 rounded-control px-3 py-3 outline-offset-[-2px] transition-colors duration-150 motion-reduce:transition-none",
    unread ? "bg-accent-wash/45 hover:bg-accent-wash/80" : "hover:bg-deep",
  );
  const body = (
    <>
      <span
        className={cn(
          "mt-0.5 grid size-9 shrink-0 place-items-center rounded-control",
          mark.tone,
          !unread && "opacity-70",
        )}
      >
        <Icon aria-hidden="true" className="size-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span
          className={cn(
            "block text-ui leading-snug wrap-anywhere",
            unread ? "text-ink" : "text-mute",
          )}
        >
          {words.lead}{" "}
          <span
            className={cn("font-medium", unread ? "text-ink" : "text-ink/80")}
          >
            {words.subject}
          </span>
        </span>
        {words.detail ? (
          <span className="mt-1 block font-prose text-meta text-mute wrap-anywhere">
            {words.detail}
          </span>
        ) : null}
        <time
          className="mt-1.5 block text-label text-mute tabular-nums"
          dateTime={entry.createdAt}
          title={readableMoment(entry.createdAt)}
        >
          {arrivedAgo(entry.createdAt, now)}
        </time>
      </span>
      {unread ? (
        <span className="mt-2 flex shrink-0 items-start">
          <span aria-hidden="true" className="size-2 rounded-full bg-accent" />
          <span className="sr-only">Unread</span>
        </span>
      ) : null}
    </>
  );

  return (
    <li>
      {words.href ? (
        <Link href={words.href} onClick={() => onOpen(entry)} className={row}>
          {body}
        </Link>
      ) : (
        <div className={row}>{body}</div>
      )}
    </li>
  );
}

/** Stands in for an entry while the inbox loads. */
export function NotificationEntrySkeleton() {
  return (
    <li aria-hidden="true" className="flex gap-3.5 px-3 py-3">
      <span className="size-9 shrink-0 animate-pulse rounded-control bg-deep" />
      <span className="flex-1">
        <span className="block h-3.5 w-3/4 animate-pulse rounded-full bg-deep" />
        <span className="mt-2.5 block h-3 w-1/2 animate-pulse rounded-full bg-deep" />
        <span className="mt-2.5 block h-2.5 w-16 animate-pulse rounded-full bg-deep" />
      </span>
    </li>
  );
}
