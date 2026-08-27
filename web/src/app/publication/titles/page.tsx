import { Shell } from "@/components/layout/Shell";
import { TitlesAndBadges } from "@/components/publication/TitlesAndBadges";
import { pageMetadata } from "@/lib/site-metadata";
import styles from "./TitlesPage.module.css";

export const metadata = pageMetadata(
  "Titles and badges",
  "Define the Illarin positions, titles and badges that appear on public profiles, and give them to accounts.",
);

export default function TitlesAndBadgesPage() {
  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.heading}>
          <h1>Titles and badges</h1>
          <p>
            These appear on a person's public profile. None of them grants any
            permission on Illarin.
          </p>
        </header>
        <TitlesAndBadges />
      </Shell>
    </section>
  );
}
