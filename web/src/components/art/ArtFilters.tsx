export function ArtFilters() {
  return (
    <svg
      width="0"
      height="0"
      aria-hidden="true"
      focusable="false"
      className="absolute"
    >
      <filter id="watercolor-dark-ink" colorInterpolationFilters="sRGB">
        <feFlood floodColor="var(--v-action)" result="ink" />
        <feComposite in="ink" in2="SourceAlpha" operator="in" />
      </filter>
    </svg>
  );
}
