import type { RegisterCell } from "@/components/register/RegisterRail";
import type {
  Distinction,
  DistinctionAssignment,
  DistinctionForm,
} from "@/lib/api/query";
import { moved } from "@/lib/reorder";

export type RecognitionRegister = "titles" | "positions" | "accounts";

export const RECOGNITION_REGISTERS: RecognitionRegister[] = [
  "titles",
  "positions",
  "accounts",
];

const NAMES: Record<RecognitionRegister, string> = {
  accounts: "Account recognition",
  positions: "Illarin positions",
  titles: "Titles and badges",
};

const FORMS: Record<RecognitionRegister, DistinctionForm[]> = {
  accounts: [],
  positions: ["position"],
  titles: ["badge", "title"],
};

export function registerName(register: RecognitionRegister): string {
  return NAMES[register];
}

export function recognitionCells(
  definitions: Distinction[],
): RegisterCell<RecognitionRegister>[] {
  return RECOGNITION_REGISTERS.map((register) => ({
    count:
      register === "accounts"
        ? null
        : registerHolds(register, definitions).current.length,
    id: register,
    name: NAMES[register],
  }));
}

export function registerHolds(
  register: RecognitionRegister,
  definitions: Distinction[],
): { current: Distinction[]; retired: Distinction[] } {
  const forms = FORMS[register];
  const mine = definitions.filter((one) => forms.includes(one.form));
  return {
    current: mine
      .filter((one) => !one.retired)
      .sort((a, b) => forms.indexOf(a.form) - forms.indexOf(b.form)),
    retired: mine.filter((one) => one.retired),
  };
}

export function nothingIn(register: RecognitionRegister): string {
  if (register === "titles") {
    return "No titles or badges defined. Add an image to create a badge, or leave it out to create a title.";
  }
  if (register === "positions") {
    return "No positions defined. Add an Illarin position to show a person's role on their profile. It grants no permissions.";
  }
  return "Look up an account to manage its positions, titles and badges.";
}

export function unexplained(one: Distinction): boolean {
  return one.form !== "position" && one.explanation.trim() === "";
}

export function movedDefinitions(
  definitions: Distinction[],
  one: Distinction,
  step: number,
): string[] | null {
  const register = one.form === "position" ? "positions" : "titles";
  const peers = registerHolds(register, definitions).current.filter(
    (other) => other.form === one.form,
  );
  const at = peers.indexOf(one);
  const target = at + step;
  if (at < 0 || target < 0 || target >= peers.length) return null;
  const family = definitions.filter((other) => other.form === one.form);
  return moved(
    family.map((other) => other.id),
    family.indexOf(peers[at]),
    family.indexOf(peers[target]),
  );
}

export function heldBy(assignments: DistinctionAssignment[]): {
  earned: DistinctionAssignment[];
  past: DistinctionAssignment[];
  positions: DistinctionAssignment[];
} {
  const active = assignments.filter((one) => one.active);
  return {
    earned: active.filter((one) => one.distinction.form !== "position"),
    past: assignments.filter((one) => !one.active),
    positions: active.filter((one) => one.distinction.form === "position"),
  };
}

export function whatIsHeld(assignments: DistinctionAssignment[]): string {
  const { earned, positions } = heldBy(assignments);
  if (earned.length === 0 && positions.length === 0) {
    return "No recognition assigned";
  }
  const jobs = `${positions.length} ${positions.length === 1 ? "job" : "jobs"}`;
  return `${jobs} · ${earned.length} earned`;
}

export function offeredTo(
  definitions: Distinction[],
  assignments: DistinctionAssignment[],
): Distinction[] {
  const { earned, positions } = heldBy(assignments);
  const held = new Set(
    [...positions, ...earned].map((one) => one.distinction.id),
  );
  return definitions.filter((one) => !one.retired && !held.has(one.id));
}

export function movedAssignments(
  assignments: DistinctionAssignment[],
  one: DistinctionAssignment,
  step: number,
): string[] | null {
  const { earned } = heldBy(assignments);
  const at = earned.indexOf(one);
  const target = at + step;
  if (at < 0 || target < 0 || target >= earned.length) return null;
  const order = assignments
    .filter((other) => other.active)
    .map((other) => other.id);
  return moved(
    order,
    order.indexOf(earned[at].id),
    order.indexOf(earned[target].id),
  );
}
