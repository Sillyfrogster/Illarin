import type { ReactNode } from "react";
import type { RecordedVersion } from "@/lib/api/query";
import { VersionDownload } from "./VersionDownload";
import { VersionEntry } from "./VersionEntry";
import { VersionSpine } from "./VersionSpine";

export function UpdateHistory({
  assetId,
  kind,
  versions,
  download,
  olderDownloads,
}: {
  assetId: string;
  kind: string;
  versions: RecordedVersion[];
  download: ReactNode;
  olderDownloads: boolean;
}) {
  if (versions.length === 0) {
    return (
      <p className="mt-group max-w-[60ch] font-prose text-prose text-mute">
        Illarin has recorded no versions of this {kind} yet. History begins at
        its first publication.
      </p>
    );
  }

  return (
    <div className="mt-group grid items-start gap-x-12 gap-y-8 lg:grid-cols-[13rem_minmax(0,1fr)]">
      <aside className="lg:sticky lg:top-[calc(var(--header-height)+2rem)]">
        <VersionSpine versions={versions} />
      </aside>
      <ol className="min-w-0 list-none">
        {versions.map((version, index) => (
          <VersionEntry
            assetId={assetId}
            current={index === 0}
            download={
              index === 0 ? (
                download
              ) : olderDownloads ? (
                <VersionDownload
                  assetId={assetId}
                  kind={kind}
                  version={version}
                />
              ) : null
            }
            key={version.id}
            kind={kind}
            version={version}
            versions={versions}
          />
        ))}
      </ol>
    </div>
  );
}
