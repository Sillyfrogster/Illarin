import {
  CircleUserRound,
  LogIn,
  type LucideIcon,
  Mail,
  NotebookPen,
  Settings,
  UserPlus,
} from "lucide-react";
import type { AccountDestination } from "./destinations";

const icons: Record<AccountDestination["id"], LucideIcon> = {
  profile: CircleUserRound,
  settings: Settings,
  publication: NotebookPen,
  verify: Mail,
  "sign-in": LogIn,
  "sign-up": UserPlus,
};

export function DestinationIcon({ id }: { id: AccountDestination["id"] }) {
  const Icon = icons[id];
  return <Icon aria-hidden="true" className="size-4 shrink-0 text-mute" />;
}
