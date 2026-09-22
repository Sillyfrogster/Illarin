"use client";

/** DiscordChoice asks whether a post's first publication goes to the blog's Discord channel. */
export function DiscordChoice({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex min-h-11 cursor-pointer items-start gap-3 rounded-control bg-deep p-3 font-ui text-ui text-ink">
      <input
        checked={checked}
        className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span className="min-w-0">
        Post to Discord
        <span className="mt-1 block font-prose text-meta text-mute">
          The title, summary and a link go to the blog's Discord channel, if an
          admin connected one.
        </span>
      </span>
    </label>
  );
}
