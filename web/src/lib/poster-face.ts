/** A catalog poster either shows the creator's picture or sets their title in type. */
export type PosterFace = "art" | "type";

/** How large a type poster can set a name before it stops fitting its plate. */
export type TypeSetting = "grand" | "large" | "medium" | "small";

/** What the plate holds at each setting, as characters across then lines down. */
const PLATE: Record<Exclude<TypeSetting, "small">, [number, number]> = {
  grand: [9, 2],
  large: [13, 3],
  medium: [18, 4],
};

export function posterFace({
  cover,
  failed = false,
}: {
  cover: unknown | null;
  failed?: boolean;
}): PosterFace {
  return cover && !failed ? "art" : "type";
}

/**
 * The largest setting whose plate holds the whole name, judged by its length
 * and by its longest unbreakable run, because one long word overruns a plate
 * that the character count says would fit. A line can break at a space, a
 * hyphen, a slash or an underscore, so those end a run.
 */
export function typeSetting(name: string): TypeSetting {
  const written = name.trim();
  if (written === "") return "grand";

  const longestRun = Math.max(
    ...written.split(/[\s\-_/·—–]+/).map((run) => run.length),
  );

  for (const setting of ["grand", "large", "medium"] as const) {
    const [across, down] = PLATE[setting];
    if (written.length <= across * down && longestRun <= across) {
      return setting;
    }
  }
  return "small";
}
