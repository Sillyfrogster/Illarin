/** The band a post title falls in, so its display type is sized to the sentence. */
export function titleBand(title: string): "short" | "medium" | "long" {
  if (title.length > 78) return "long";
  if (title.length > 42) return "medium";
  return "short";
}
