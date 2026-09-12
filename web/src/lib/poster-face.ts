export type PosterFace = "art" | "type";

export type TypeSetting = "grand" | "large" | "medium" | "small";

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
