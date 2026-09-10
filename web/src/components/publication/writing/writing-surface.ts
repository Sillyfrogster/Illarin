const FLOW =
  "min-h-[26rem] font-prose text-article break-words text-ink outline-none [&>*+*]:mt-5";

const HEADINGS = [
  "[&_h2]:mt-12 [&_h2]:mb-4 [&_h2]:font-display [&_h2]:text-title [&_h2]:leading-tight [&_h2]:font-medium [&_h2]:text-balance",
  "[&_h3]:mt-10 [&_h3]:mb-3 [&_h3]:font-display [&_h3]:text-section [&_h3]:leading-snug [&_h3]:font-medium [&_h3]:text-balance",
  "[&_h4]:mt-8 [&_h4]:mb-2 [&_h4]:font-display [&_h4]:text-ui [&_h4]:font-semibold [&_h4]:tracking-[0.02em] [&_h4]:uppercase",
].join(" ");

const LISTS = [
  "[&_ul]:my-6 [&_ul]:list-disc [&_ul]:pl-6 [&_ol]:my-6 [&_ol]:list-decimal [&_ol]:pl-6",
  "[&_li+li]:mt-3 [&_li]:pl-1 [&_ul]:marker:text-mute [&_ol]:marker:text-mute",
].join(" ");

const TASKS = [
  "[&_ul[data-type=taskList]]:list-none [&_ul[data-type=taskList]]:pl-0",
  "[&_ul[data-type=taskList]>li]:flex [&_ul[data-type=taskList]>li]:items-start [&_ul[data-type=taskList]>li]:gap-3 [&_ul[data-type=taskList]>li]:pl-0",
  "[&_ul[data-type=taskList]>li>label]:mt-1.5 [&_ul[data-type=taskList]>li>label]:flex [&_ul[data-type=taskList]>li>label]:shrink-0",
  "[&_ul[data-type=taskList]>li>div]:min-w-0 [&_ul[data-type=taskList]>li>div]:flex-1",
  "[&_input[type=checkbox]]:size-5 [&_input[type=checkbox]]:accent-[var(--v-action)]",
].join(" ");

const QUOTE =
  "[&_blockquote]:my-10 [&_blockquote]:font-display [&_blockquote]:text-title [&_blockquote]:leading-snug [&_blockquote]:text-accent [&_blockquote>*+*]:mt-4";

const CODE = [
  "[&_code]:rounded-[6px] [&_code]:bg-deep [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.88em]",
  "[&_pre]:my-8 [&_pre]:overflow-x-auto [&_pre]:rounded-plate [&_pre]:bg-deep [&_pre]:p-5",
  "[&_pre_code]:block [&_pre_code]:min-w-max [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_pre_code]:text-meta [&_pre_code]:leading-7 [&_pre_code]:whitespace-pre",
].join(" ");

const TABLE = [
  "[&_.tableWrapper]:my-8 [&_.tableWrapper]:overflow-x-auto [&_.tableWrapper]:rounded-plate",
  "[&_table]:w-full [&_table]:min-w-[32rem] [&_table]:border-collapse [&_table]:text-left [&_table]:text-meta",
  "[&_th]:min-w-[10ch] [&_th]:bg-deep [&_th]:px-4 [&_th]:py-3 [&_th]:align-top [&_th]:font-medium [&_th]:text-ink",
  "[&_td]:min-w-[10ch] [&_td]:px-4 [&_td]:py-3 [&_td]:align-top",
  "[&_.selectedCell]:relative [&_.selectedCell]:after:pointer-events-none [&_.selectedCell]:after:absolute [&_.selectedCell]:after:inset-0 [&_.selectedCell]:after:bg-accent/12 [&_.selectedCell]:after:content-['']",
].join(" ");

const CALLOUT = [
  "[&_div[data-kind]]:my-8 [&_div[data-kind]]:rounded-plate [&_div[data-kind]]:bg-deep [&_div[data-kind]]:p-5",
  "[&_div[data-kind]>*+*]:mt-3 [&_div[data-kind]>*]:text-prose",
  "[&_div[data-kind=important]]:bg-accent-wash [&_div[data-kind=warning]]:bg-stop-wash",
  "[&_div[data-kind]]:before:mb-2 [&_div[data-kind]]:before:block [&_div[data-kind]]:before:font-ui [&_div[data-kind]]:before:text-meta [&_div[data-kind]]:before:font-semibold [&_div[data-kind]]:before:tracking-[0.02em] [&_div[data-kind]]:before:capitalize [&_div[data-kind]]:before:content-[attr(data-kind)]",
  "[&_div[data-kind=important]]:before:text-accent [&_div[data-kind=warning]]:before:text-stop",
].join(" ");

const MARKS = [
  "[&_a]:text-ink [&_a]:underline [&_a]:decoration-accent/55 [&_a]:underline-offset-[3px]",
  "[&_s]:text-mute",
  "[&_hr]:my-10 [&_hr]:h-px [&_hr]:w-24 [&_hr]:border-0 [&_hr]:bg-rule",
].join(" ");

const WRITING_MARKS = [
  "[&_p.is-editor-empty:first-child]:before:pointer-events-none [&_p.is-editor-empty:first-child]:before:float-left [&_p.is-editor-empty:first-child]:before:h-0 [&_p.is-editor-empty:first-child]:before:text-mute [&_p.is-editor-empty:first-child]:before:content-[attr(data-placeholder)]",
  "[&_.ProseMirror-selectednode]:outline-2 [&_.ProseMirror-selectednode]:outline-offset-2 [&_.ProseMirror-selectednode]:outline-accent",
  "[&_.ProseMirror-selectednode_img]:outline-2 [&_.ProseMirror-selectednode_img]:outline-offset-2 [&_.ProseMirror-selectednode_img]:outline-accent",
  "[&_img[alt='']]:outline-2 [&_img[alt='']]:outline-offset-2 [&_img[alt='']]:outline-stop",
].join(" ");

export const WRITING_SURFACE = [
  FLOW,
  HEADINGS,
  LISTS,
  TASKS,
  QUOTE,
  CODE,
  TABLE,
  CALLOUT,
  MARKS,
  WRITING_MARKS,
].join(" ");

export const WRITING_PICTURE =
  "my-9 [&>img]:h-auto [&>img]:w-full [&>img]:rounded-plate [&>img]:bg-deep [&>figcaption]:mt-3 [&>figcaption]:font-prose [&>figcaption]:text-meta [&>figcaption]:leading-6 [&>figcaption]:text-mute";

export const WRITING_GALLERY = "my-9 grid w-full grid-cols-2 gap-3 sm:gap-4";

export const WRITING_GALLERY_PICTURE =
  "min-w-0 [&>img]:aspect-[4/3] [&>img]:h-auto [&>img]:w-full [&>img]:rounded-plate [&>img]:bg-deep [&>img]:object-contain [&>figcaption]:mt-2 [&>figcaption]:font-prose [&>figcaption]:text-meta [&>figcaption]:leading-6 [&>figcaption]:text-mute";
