import { Badge } from "@/components/ui/badge";

export function DeclaredValues({
  label,
  values,
}: {
  label: string;
  values: string[];
}) {
  return (
    <div className="min-w-0">
      <dt className="font-ui text-meta text-mute">{label}</dt>
      <dd className="mt-1">
        {values.length > 0 ? (
          <ul className="m-0 flex list-none flex-wrap gap-1.5 p-0">
            {values.map((value) => (
              <li key={value}>
                <Badge className="font-mono font-normal">{value}</Badge>
              </li>
            ))}
          </ul>
        ) : (
          <span className="font-ui text-ui text-mute">None declared</span>
        )}
      </dd>
    </div>
  );
}
