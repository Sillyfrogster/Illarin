import {
  LockKeyhole,
  LockKeyholeOpen,
  type LucideIcon,
  ShieldCheck,
  ShieldOff,
  Sparkles,
  X,
} from "lucide-react";
import Link from "next/link";
import type { Notification } from "@/lib/api/notifications";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { arrivedAgo, notificationWords } from "@/lib/notification-inbox";
import { SendUpdates } from "./SendUpdates";

const TAKEN_TONE = "bg-stop-wash text-stop";
const GIVEN_BACK_TONE = "bg-accent-wash text-accent";
const NEWS_TONE = "bg-deep text-ink";

const MARKS = {
  work_withheld: { icon: LockKeyhole, tone: TAKEN_TONE },
  work_restored: { icon: LockKeyholeOpen, tone: GIVEN_BACK_TONE },
  work_updated: { icon: Sparkles, tone: NEWS_TONE },
  profile_restricted: { icon: ShieldOff, tone: TAKEN_TONE },
  profile_restored: { icon: ShieldCheck, tone: GIVEN_BACK_TONE },
} satisfies Record<Notification["type"], { icon: LucideIcon; tone: string }>;

/** One inbox entry, tinted while unread, that opens what it is about and can be removed. */
export function NotificationEntry({
  entry,
  now,
  onOpen,
  onRemove,
}: {
  entry: Notification;
  now: Date;
  onOpen: (entry: Notification) => void;
  onRemove: (entry: Notification) => void;
}) {
  const words = notificationWords(entry);
  const unread = !entry.readAt;
  const mark = MARKS[entry.type];
  const Icon = mark.icon;
  const work = entry.work;
  const sends = entry.sendTargets ?? [];
  const row = cn(
    "flex gap-3.5 pt-3 pr-11 pl-3 outline-offset-[-2px]",
    sends.length > 0 ? "pb-1.5" : "pb-3",
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
    <li
      className={cn(
        "group/entry relative rounded-control transition-colors duration-150 motion-reduce:transition-none",
        unread ? "bg-accent-wash/45 hover:bg-accent-wash/80" : "hover:bg-deep",
      )}
    >
      {words.href ? (
        <Link href={words.href} onClick={() => onOpen(entry)} className={row}>
          {body}
        </Link>
      ) : (
        <div className={row}>{body}</div>
      )}
      {work && sends.length > 0 ? (
        <SendUpdates workId={work.id} targets={sends} />
      ) : null}
      <button
        type="button"
        title="Remove"
        onClick={() => onRemove(entry)}
        className="absolute top-2 right-2 inline-flex size-8 items-center justify-center rounded-control text-mute opacity-0 outline-offset-2 transition-[opacity,background-color,color] duration-200 group-hover/entry:opacity-100 hover:bg-plane hover:text-ink focus-visible:opacity-100 pointer-coarse:opacity-100 motion-reduce:transition-none"
      >
        <X aria-hidden="true" className="size-4" />
        <span className="sr-only">Remove this notification</span>
      </button>
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
