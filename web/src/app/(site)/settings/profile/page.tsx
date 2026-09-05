import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { PublicProfileEditor } from "@/components/profile/PublicProfileEditor";
import styles from "./ProfileSettingsPage.module.css";

export const metadata = { title: "Public profile" };

export default function PublicProfileSettings() {
  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.heading}>
          <Link className={styles.back} href="/settings">
            <ChevronLeft size={15} strokeWidth={1.8} aria-hidden="true" />
            Account settings
          </Link>
          <h1>Public profile</h1>
          <p>
            Everything here is visible to anyone who opens your profile. Your
            handle stays your address; leave a field empty to show nothing.
          </p>
        </header>
        <PublicProfileEditor />
      </Shell>
    </section>
  );
}
