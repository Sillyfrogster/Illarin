import type { NsfwPreference } from "@/lib/api/query";

const PREFERENCE_KEY = "illarin.nsfw-visibility";
const REVEAL_KEY = "illarin.nsfw-reveal:v1";

export function readSessionPreference(): NsfwPreference | undefined {
  try {
    const stored = window.sessionStorage.getItem(PREFERENCE_KEY);
    return isPreference(stored) ? stored : undefined;
  } catch {
    return undefined;
  }
}

export function writeSessionPreference(preference: NsfwPreference) {
  try {
    window.sessionStorage.setItem(PREFERENCE_KEY, preference);
  } catch {}
}

export function readWorkReveal(id: string) {
  try {
    return window.sessionStorage.getItem(`${REVEAL_KEY}:${id}`) === "1";
  } catch {
    return false;
  }
}

export function writeWorkReveal(id: string) {
  try {
    window.sessionStorage.setItem(`${REVEAL_KEY}:${id}`, "1");
  } catch {}
}

function isPreference(value: string | null): value is NsfwPreference {
  return value === "hidden" || value === "blurred" || value === "shown";
}
