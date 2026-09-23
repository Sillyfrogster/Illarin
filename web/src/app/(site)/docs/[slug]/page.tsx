import Link from "next/link";
import { notFound } from "next/navigation";
import { DocMarkdown, docSections } from "@/components/docs/DocMarkdown";
import { Shell } from "@/components/layout/Shell";
import { DOCS, findDoc, readDoc } from "@/lib/docs";
import { pageMetadata } from "@/lib/site-metadata";

export function generateStaticParams() {
  return DOCS.map(({ slug }) => ({ slug }));
}

export async function generateMetadata({ params }: PageProps<"/docs/[slug]">) {
  const { slug } = await params;
  const doc = findDoc(slug);
  return doc ? pageMetadata(doc.title, doc.description) : {};
}

export default async function DocPage({ params }: PageProps<"/docs/[slug]">) {
  const { slug } = await params;
  const doc = findDoc(slug);
  if (!doc) notFound();
  const source = await readDoc(doc.slug);
  const sections = docSections(source);
  return (
    <Shell className="pt-8 pb-chapter lg:pt-12">
      <div className="grid min-w-0 gap-10 lg:grid-cols-[13.5rem_minmax(0,1fr)_12rem] lg:gap-10">
        <aside className="lg:sticky lg:top-28 lg:self-start">
          <Link
            className="flex min-h-11 items-center font-ui text-meta text-mute hover:text-accent"
            href="/docs"
          >
            ← All documentation
          </Link>
          <nav
            aria-label="Documentation guides"
            className="mt-4 border-t border-rule pt-4"
          >
            <p className="font-ui text-label font-medium text-mute">Guides</p>
            {DOCS.map((entry) => (
              <Link
                aria-current={entry.slug === slug ? "page" : undefined}
                className="flex min-h-11 items-center rounded-control px-3 font-ui text-ui text-mute hover:bg-deep hover:text-ink aria-[current=page]:bg-deep aria-[current=page]:font-medium aria-[current=page]:text-ink"
                href={`/docs/${entry.slug}`}
                key={entry.slug}
              >
                {entry.title}
              </Link>
            ))}
          </nav>
        </aside>
        <article className="min-w-0 max-w-[75ch]">
          <details className="mb-8 rounded-control bg-deep px-4 lg:hidden">
            <summary className="flex min-h-11 cursor-pointer items-center font-ui text-ui text-ink">
              On this page
            </summary>
            <nav aria-label="On this page" className="grid pb-3">
              {sections.map((section) => (
                <a
                  className="flex min-h-11 items-center font-ui text-ui text-mute hover:text-accent"
                  href={`#${section.id}`}
                  key={section.id}
                >
                  {section.title}
                </a>
              ))}
            </nav>
          </details>
          <DocMarkdown source={source} />
        </article>
        <aside className="hidden lg:sticky lg:top-28 lg:block lg:max-h-[calc(100vh-8rem)] lg:self-start lg:overflow-y-auto">
          <nav aria-label="On this page">
            <p className="font-ui text-label font-medium text-mute">
              On this page
            </p>
            <ul className="mt-3 border-l border-rule">
              {sections.map((section) => (
                <li key={section.id}>
                  <a
                    className="block py-1.5 pl-3 font-ui text-meta text-mute hover:border-l-2 hover:border-accent hover:pl-[10px] hover:text-ink"
                    href={`#${section.id}`}
                  >
                    {section.title}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        </aside>
      </div>
    </Shell>
  );
}
