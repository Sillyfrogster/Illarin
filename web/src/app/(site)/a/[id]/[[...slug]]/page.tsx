import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { cache } from "react";
import { shellClasses } from "@/components/layout/Shell";
import { fetchWork, type WorkDetail } from "@/lib/api/query";
import { DraftedChangesProvider } from "@/lib/drafted-changes";
import { ExtensionDependenciesProvider } from "@/lib/extension-dependencies";
import { readableForMetadata } from "@/lib/site-metadata";
import { workMetadata } from "@/lib/work-metadata";
import { workHoldsNothing } from "@/lib/work-page-content";
import { TYPE_LABELS } from "@/lib/work-types";
import { isWorkId, workRedirect } from "@/lib/work-url";
import { WorkBlocks } from "./WorkBlocks";
import { WorkHeader } from "./WorkHeader";
import { WorkspaceProvider } from "./workspace/state";
import { WorkspaceSurfaces } from "./workspace/WorkspaceSurfaces";

const loadWork = cache(async (id: string): Promise<WorkDetail | null> => {
  if (!isWorkId(id)) return null;
  const cookie = (await cookies()).toString();
  return fetchWork(id, cookie);
});

export async function generateMetadata({
  params,
}: PageProps<"/a/[id]/[[...slug]]">): Promise<Metadata> {
  const work = await readableForMetadata(loadWork((await params).id));
  return work ? workMetadata(work) : { title: "Not found" };
}

export default async function WorkPage({
  params,
}: PageProps<"/a/[id]/[[...slug]]">) {
  const { id, slug } = await params;
  const published = await loadWork(id);
  if (!published) notFound();

  const canonical = workRedirect({ id, slug }, published);
  if (canonical) redirect(canonical);

  const work = published.isOwner
    ? await fetchWork(id, (await cookies()).toString(), true)
    : published;
  if (!work) notFound();

  const typeLabel = TYPE_LABELS[work.type];
  const isDraft = work.lifecycle === "draft";
  const sharedDate = new Date(work.createdAt).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });

  return (
    <DraftedChangesProvider key={work.id} version={work.draftedChangesVersion}>
      <WorkspaceProvider
        addableBlocks={work.addableBlocks ?? []}
        allowedApps={work.allowedApps}
        workId={work.id}
        blocks={work.blocks}
        eligibleApps={work.eligibleApps}
        details={{
          blurb: work.blurb,
          isNsfw: work.isNsfw,
          name: work.name,
        }}
        isDraft={isDraft}
        isOwner={work.isOwner}
        unpublishedChanges={Boolean(work.unpublishedChanges)}
      >
        <ExtensionDependenciesProvider
          dependencies={work.extensionDependencies}
        >
          <div className="relative isolate overflow-x-clip pb-chapter">
            <article>
              <WorkHeader
                work={work}
                holdsNothing={workHoldsNothing(work.blocks)}
                typeLabel={typeLabel}
                sharedDate={sharedDate}
                shellClassName={shellClasses}
              />
              <WorkBlocks
                images={work.media}
                isOwner={work.isOwner}
                type={work.type}
                shellClassName={shellClasses}
              />
            </article>
          </div>
        </ExtensionDependenciesProvider>
        <WorkspaceSurfaces
          creator={work.creator}
          visibility={work.visibility}
          hasOriginal={Boolean(work.original)}
          images={work.media}
          typeName={typeLabel.toLowerCase()}
          readiness={work.readiness}
          preservedPrompts={work.preservedPrompts}
          hasPrivatePrompts={work.hasPrivatePrompts}
          unpublishedChanges={Boolean(work.unpublishedChanges)}
          withheld={Boolean(work.withhold)}
        />
      </WorkspaceProvider>
    </DraftedChangesProvider>
  );
}
