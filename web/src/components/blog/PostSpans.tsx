import type { PostSpan } from "@/lib/post-body";
import { PostLink } from "./PostLink";

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
        rendered = <PostLink href={mark.href}>{rendered}</PostLink>;
        break;
    }
  }
  return rendered;
}
