import { Shell } from "@/components/layout/Shell";
import { ContributorWorkspace } from "@/components/publication/ContributorWorkspace";
import { pageMetadata } from "@/lib/site-metadata";
import styles from "./WorkspacePage.module.css";

export const metadata = pageMetadata(
  "Publication workspace",
  "The app, categories and default you are approved to publish under.",
);

export default function PublicationWorkspacePage() {
  return (
    <section className={styles.page}>
      <Shell className={styles.layout}>
        <header className={styles.masthead}>
          <p className={styles.eyebrow}>Illarin publication</p>
          <h1>Your workspace</h1>
        </header>
        <ContributorWorkspace />
      </Shell>
    </section>
  );
}
