import type { BlogCategory, WriterResponse } from "@/lib/api/query";

export type Register = "writers" | "categories" | "discord";

export const REGISTERS: Register[] = ["writers", "categories", "discord"];

const NAMES: Record<Register, string> = {
  writers: "Writers",
  categories: "Categories",
  discord: "Discord",
};

export function registerName(register: Register): string {
  return NAMES[register];
}

export type RegisterStanding = { count: number | null; attention: boolean };

export function registerStandings(held: {
  writers: WriterResponse[];
  categories: BlogCategory[];
}): Record<Register, RegisterStanding> {
  return {
    writers: kept(held.writers.length),
    categories: kept(held.categories.filter((one) => !one.retired).length),
    discord: { attention: false, count: null },
  };
}

function kept(count: number): RegisterStanding {
  return { attention: false, count };
}

export function nothingIn(register: "writers" | "categories"): string {
  if (register === "writers") {
    return "No account has the writer switch on. Admins can still publish.";
  }
  return "The blog has no categories, so no post can be filed.";
}
