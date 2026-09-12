const UNITS = [
  { scale: 1024 ** 3, suffix: "GB" },
  { scale: 1024 ** 2, suffix: "MB" },
  { scale: 1024, suffix: "KB" },
];

export function fileWeight(bytes: number): string {
  for (const unit of UNITS) {
    if (bytes < unit.scale) continue;
    const size = (bytes / unit.scale).toFixed(1).replace(/\.0$/, "");
    return `${size} ${unit.suffix}`;
  }
  return bytes === 1 ? "1 byte" : `${bytes} bytes`;
}
