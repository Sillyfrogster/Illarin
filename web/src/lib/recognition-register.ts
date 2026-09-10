import type { RegisterCell } from "@/components/register/RegisterRail";
import type {
  Distinction,
  DistinctionAssignment,
  DistinctionForm,
} from "@/lib/api/query";
import { moved } from "@/lib/reorder";

/** One of the things Illarin gives out, or the account it gives them to. */
export type RecognitionRegister = "titles" | "positions" | "accounts";

/** The order the registers are offered in, everywhere they are offered. */
export const RECOGNITION_REGISTERS: RecognitionRegister[] = [
  "titles",
  "positions",
  "accounts",
];

const NAMES: Record<RecognitionRegister, string> = {
  accounts: "Someone's profile",
  positions: "Illarin positions",
  titles: "Titles and badges",
};

/** A badge is a title with a mark, so one register holds both, marks first. */
const FORMS: Record<RecognitionRegister, DistinctionForm[]> = {
  accounts: [],
  positions: ["position"],
  titles: ["badge", "title"],
};

export function registerName(register: RecognitionRegister): string {
  return NAMES[register];
}

/** What each register carries, for the rail that chooses between them. */
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

/** What a register still gives out, and what it has stopped giving out. */
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

/** What an empty register says, which is what to do about it rather than that it is empty. */
export function nothingIn(register: RecognitionRegister): string {
  if (register === "titles") {
    return "Nothing is defined yet. Give one a mark and it shows as a badge; leave the mark off and it shows as a title.";
  }
  if (register === "positions") {
    return "No position is defined. Illarin's own jobs go here, and holding one lets nobody do anything.";
  }
  return "Look up a handle to see what that account holds.";
}

/** A title or badge that says nothing about what earns it leaves a profile reader guessing. */
export function unexplained(one: Distinction): boolean {
  return one.form !== "position" && one.explanation.trim() === "";
}

/**
 * The new order for one form when a definition moves a step through it. The
 * retired ones come along, because the server orders the whole form at once.
 */
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

/** An account's jobs, what it earned, and what was taken back. */
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

/** One line saying what an account carries on its profile. */
export function whatIsHeld(assignments: DistinctionAssignment[]): string {
  const { earned, positions } = heldBy(assignments);
  if (earned.length === 0 && positions.length === 0) {
    return "Nothing on their profile yet";
  }
  const jobs = `${positions.length} ${positions.length === 1 ? "job" : "jobs"}`;
  return `${jobs} · ${earned.length} earned`;
}

/** What is left to give this account, which is everything still given out that they do not hold. */
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

/**
 * The new order for one account when an earned title moves a step. Positions
 * keep their places in it, because the server orders every assignment at once.
 */
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
