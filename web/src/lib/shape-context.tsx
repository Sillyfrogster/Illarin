/** ShapeClasses are the corner radii every control shares, at the site's 10px control radius and 14px plate radius. */
interface ShapeClasses {
  item: string;
  bg: string;
  focusRing: string;
  mergedBg: string;
  container: string;
  button: string;
  input: string;
  bgRadius: number;
  containerRadius: number;
  mergedRadius: number;
}

const rounded: ShapeClasses = {
  item: "rounded-control",
  bg: "rounded-control",
  focusRing: "rounded-[12px]",
  mergedBg: "rounded-control",
  container: "rounded-plate",
  button: "rounded-control",
  input: "rounded-control",
  bgRadius: 10,
  containerRadius: 14,
  mergedRadius: 10,
};

const shapeMap = { rounded };

/** useShape returns the one shape the site uses; it keeps Fluid's hook name so its components read unchanged. */
function useShape(): ShapeClasses {
  return rounded;
}

/** nestedRadius is the radius for a surface inset inside another rounded surface. */
function nestedRadius(outerRadius: number, inset: number, borderWidth = 0) {
  return Math.max(0, outerRadius - inset - borderWidth);
}

export { nestedRadius, shapeMap, useShape };
export type { ShapeClasses };
