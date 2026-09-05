/** The date and time halves a pair of native inputs holds. */
export type LocalParts = { date: string; time: string };

/** The instant a local date and time name, carrying the reader's offset. */
export function toInstant(date: string, time: string): string {
  if (!date || !time) return "";
  const moment = new Date(`${date}T${time}`);
  if (Number.isNaN(moment.getTime())) return "";
  return `${date}T${time}:00${offsetOf(moment)}`;
}

/** The date and time inputs show for an instant, in the reader's own zone. */
export function localParts(instant: string): LocalParts {
  const moment = new Date(instant);
  if (Number.isNaN(moment.getTime())) return { date: "", time: "" };
  return {
    date: `${moment.getFullYear()}-${pad(moment.getMonth() + 1)}-${pad(moment.getDate())}`,
    time: `${pad(moment.getHours())}:${pad(moment.getMinutes())}`,
  };
}

/** A first suggestion far enough ahead that the worker has not passed it. */
export function atLeastAnHourAhead(from: Date = new Date()): LocalParts {
  return localParts(new Date(from.getTime() + 3600_000).toString());
}

/** The zone a time is being entered in, named the way the reader's system does. */
export function zoneLabel(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || "your time zone";
}

function offsetOf(moment: Date): string {
  const minutes = -moment.getTimezoneOffset();
  const sign = minutes < 0 ? "-" : "+";
  const size = Math.abs(minutes);
  return `${sign}${pad(Math.floor(size / 60))}:${pad(size % 60)}`;
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}
