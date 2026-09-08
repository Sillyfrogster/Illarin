import type { PostSpan } from "@/lib/post-document";
import { isSafeAddress, leavesIllarin } from "@/lib/post-link";

export function PostSpans({ spans }: { spans: PostSpan[] }) {
  return (
    <>
      {spans.map((span, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: Runs follow the writing and hold no local state.
        <Span key={index} span={span} />
      ))}
    </>
  );
}

function Span({ span }: { span: PostSpan }) {
  let rendered = <>{span.text}</>;
  for (const mark of span.marks ?? []) {
    switch (mark.type) {
      case "code":
        rendered = (
          <code className="rounded-[6px] bg-deep px-1.5 py-0.5 font-mono text-[0.88em]">
            {rendered}
          </code>
        );
        break;
      case "strike":
        rendered = <s className="text-mute">{rendered}</s>;
        break;
      case "italic":
        rendered = <em>{rendered}</em>;
        break;
      case "bold":
        rendered = <strong>{rendered}</strong>;
        break;
      case "link":
        rendered = <Away href={mark.href}>{rendered}</Away>;
        break;
    }
  }
  return rendered;
}

function Away({ href, children }: { href: string; children: React.ReactNode }) {
  if (!isSafeAddress(href)) return <>{children}</>;
  const away = leavesIllarin(href);
  return (
    <a
      className="text-ink underline decoration-accent/55 underline-offset-[3px] transition-colors hover:decoration-accent"
      href={href}
      rel="noreferrer nofollow"
      target={away ? "_blank" : undefined}
    >
      {children}
      {away ? <span className="sr-only"> (opens in a new tab)</span> : null}
    </a>
  );
}
