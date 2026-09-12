export type LocalParts = { date: string; time: string };

export function toInstant(date: string, time: string): string {
  if (!date || !time) return "";
  const moment = new Date(`${date}T${time}`);
  if (Number.isNaN(moment.getTime())) return "";
  return `${date}T${time}:00${offsetOf(moment)}`;
}

export function localParts(instant: string): LocalParts {
  const moment = new Date(instant);
  if (Number.isNaN(moment.getTime())) return { date: "", time: "" };
  return {
    date: `${moment.getFullYear()}-${pad(moment.getMonth() + 1)}-${pad(moment.getDate())}`,
    time: `${pad(moment.getHours())}:${pad(moment.getMinutes())}`,
  };
}

export function atLeastAnHourAhead(from: Date = new Date()): LocalParts {
  return localParts(new Date(from.getTime() + 3600_000).toString());
}

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

export function howSoon(instant: string, from: Date = new Date()): string {
  const moment = new Date(instant);
  if (Number.isNaN(moment.getTime())) return "";
  const away = moment.getTime() - from.getTime();
  if (away <= 0) return "";
  const minutes = Math.floor(away / 60_000);
  if (minutes < 1) return "in under a minute";
  if (minutes < 60) return count(minutes, "minute");
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return count(hours, "hour");
  return count(Math.floor(hours / 24), "day");
}

function count(size: number, unit: string): string {
  return `in ${size} ${unit}${size === 1 ? "" : "s"}`;
}
