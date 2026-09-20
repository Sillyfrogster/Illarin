import type { LucideIcon } from "lucide-react";
import {
  CalendarClock,
  CircleAlert,
  EyeOff,
  Loader,
  Trash2,
} from "lucide-react";
import type { Post } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import { type PostNotice, postNotices } from "@/lib/post-writing";

const MARKS: Record<PostNotice["kind"], LucideIcon> = {
  deleted: Trash2,
  unpublished: EyeOff,
  scheduled: CalendarClock,
};

export function PostNotices({
  onOpenPublish,
  post,
}: {
  onOpenPublish: () => void;
  post: Post;
}) {
  const notices = postNotices(post);
  if (notices.length === 0) return null;

  return (
    <div className="mt-6 flex flex-col gap-3">
      {notices.map((notice) => (
        <Notice
          key={notice.kind}
          notice={notice}
          onOpenPublish={onOpenPublish}
          post={post}
        />
      ))}
    </div>
  );
}

function Notice({
  notice,
  onOpenPublish,
  post,
}: {
  notice: PostNotice;
  onOpenPublish: () => void;
  post: Post;
}) {
  const Mark =
    notice.kind === "scheduled" && post.schedule?.state === "publishing"
      ? Loader
      : notice.kind === "scheduled" && post.schedule?.state === "stopped"
        ? CircleAlert
        : MARKS[notice.kind];

  return (
    <section
      aria-label={notice.heading}
      className={cn(
        "flex flex-wrap items-start gap-x-4 gap-y-3 rounded-plate p-4 sm:p-5",
        notice.tone === "stop" ? "bg-stop-wash" : "bg-accent-wash",
      )}
    >
      <Mark
        aria-hidden="true"
        className={cn(
          "mt-0.5 size-4 shrink-0",
          notice.tone === "stop" ? "text-stop" : "text-accent",
        )}
      />
      <div className="min-w-0 flex-1">
        <p
          className={cn(
            "font-ui text-ui font-medium",
            notice.tone === "stop" ? "text-stop" : "text-accent",
          )}
        >
          {notice.heading}
        </p>
        <p className="mt-1 font-prose text-meta text-ink">
          {notice.said}
          {notice.kind === "scheduled" && post.schedule?.state === "pending" ? (
            <>
              {" "}
              <time dateTime={post.schedule.at}>
                {readableMoment(post.schedule.at)}
              </time>
              .
            </>
          ) : null}
          {notice.kind === "deleted" && post.deletion ? (
            <span suppressHydrationWarning>
              {" "}
              {remainingDeletionWindow(post.deletion.until)}.
            </span>
          ) : null}
        </p>
        <p className="mt-1 font-prose text-meta text-mute">
          {notice.meanwhile}
        </p>
        {notice.record ? (
          <p className="mt-1 font-prose text-meta text-mute">
            Illarin's record: {notice.record}
          </p>
        ) : null}
      </div>
      <button
        className="inline-flex min-h-11 shrink-0 items-center rounded-control px-3 font-ui text-meta font-medium text-ink underline-offset-4 outline-offset-3 hover:underline"
        onClick={onOpenPublish}
        type="button"
      >
        Open publishing controls
      </button>
    </section>
  );
}
