import { LockKeyhole } from "lucide-react";
import type { AssetDetail } from "@/lib/api/query";

export function WithholdNotice({
  withhold,
}: {
  withhold: NonNullable<AssetDetail["withhold"]>;
}) {
  const recorded = new Date(withhold.at).toLocaleString("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });

  return (
    <section
      aria-labelledby="withhold-heading"
      className="mt-6 flex max-w-[42ch] gap-3 rounded-plate bg-stop-wash p-4"
    >
      <LockKeyhole aria-hidden="true" className="mt-0.5 size-5 shrink-0" />
      <div className="min-w-0">
        <h2 className="text-ui font-medium text-ink" id="withhold-heading">
          Withheld from public view
        </h2>
        <p className="mt-1 text-meta text-ink">{withhold.reason}</p>
        <p className="mt-1 text-meta text-mute">
          Recorded by @{withhold.actor} on {recorded}
        </p>
      </div>
    </section>
  );
}
