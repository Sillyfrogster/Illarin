type BrandMarkProps = {
  size?: number;
  tone?: "full" | "accent" | "faint";
};

const FILL = {
  full: "var(--v-ink)",
  accent: "var(--v-accent)",
  faint: "var(--v-mute)",
} as const;

export const MARK_PATH =
  "M46.55 .58C48.1-.35 50 .77 50 2.6V18.5C50 20.8 49.2 23.03 47.75 24.8L44.2 29.2C42.78 30.95 42 33.13 42 35.38V109.6C42 111.65 40.81 113.52 38.96 114.39L4.99 127.66C2.62 128.59 0 126.84 0 124.3V35.05C0 31.48 1.88 28.17 4.94 26.33Z";

export function BrandMark({ size = 24, tone = "full" }: BrandMarkProps) {
  const fill = FILL[tone];

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 144 144"
      fill="none"
      aria-hidden="true"
      shapeRendering="geometricPrecision"
    >
      <g fill={fill} transform="translate(16 8)">
        <path d={MARK_PATH} />
        <path d={MARK_PATH} transform="translate(112 0) scale(-1 1)" />
      </g>
    </svg>
  );
}
