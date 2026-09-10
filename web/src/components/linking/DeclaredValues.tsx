/** One thing an application or an installation says about itself, which grants nothing. */
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
              <li
                className="rounded-control bg-deep px-2.5 py-1 font-mono text-meta text-ink"
                key={value}
              >
                {value}
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
