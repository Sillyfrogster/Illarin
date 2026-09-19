import Link from "next/link";
import { CONTACT_EMAIL } from "@/lib/contact";
import { pageMetadata } from "@/lib/site-metadata";
import { type LegalClause, LegalPage } from "../LegalPage";

export const metadata = pageMetadata(
  "Terms of Service",
  "How Illarin works, what stays yours, and what you agree to by using it.",
);

const CLAUSES: LegalClause[] = [
  {
    body: (
      <>
        <p>
          Illarin is a personal project run by one person. There is no company
          behind it. It charges nothing, sells nothing, and carries no
          advertising. &ldquo;We,&rdquo; &ldquo;us,&rdquo; and
          &ldquo;Illarin&rdquo; mean the person who runs it, reachable at{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
        </p>
      </>
    ),
    heading: "1. Who we are",
  },
  {
    body: (
      <>
        <p>
          You must be at least 13 years old to use Illarin, and 18 or older to
          see or publish work marked as adult content. By using Illarin you
          confirm you meet those requirements and that the law where you live
          does not bar you from using it.
        </p>
      </>
    ),
    heading: "2. Eligibility",
  },
  {
    body: (
      <>
        <p>
          You can reach an account with an email address and password, with a
          linked Discord identity, or with both. Every account has a handle of 3
          to 32 lowercase letters, digits, dots, and underscores, and your
          profile lives at that handle.
        </p>
        <p>
          Two rules about handles and accounts are worth stating plainly. A
          handle is never reissued: once you rename or delete your account,
          nobody else can take the handle you had. And accounts never merge: if
          you sign in with a Discord identity or an email address that already
          belongs to another account, Illarin refuses rather than joining the
          two.
        </p>
        <p>
          You are responsible for what happens under your account. Keep your
          password and your linked Discord account secure. We may suspend or
          close an account that breaks these Terms or the{" "}
          <Link href="/legal/acceptable-use">Acceptable Use Policy</Link>.
        </p>
      </>
    ),
    heading: "3. Accounts",
  },
  {
    body: (
      <>
        <h3>4.1 It stays yours</h3>
        <p>
          You keep ownership of the characters, lorebooks, presets, themes, and
          packs you upload. You are responsible for them.
        </p>

        <h3>4.2 What you let us do with it</h3>
        <p>
          So that Illarin can do its job, you grant us a worldwide,
          non-exclusive, royalty-free licence to store your work, show it to the
          people you publish it to, hand it to the apps you or your readers
          connect, resize your images for display, and convert your work into
          formats for other apps. That licence covers nothing else, and it ends
          when you delete the work, apart from copies already in backups or ones
          the law requires us to keep.
        </p>
        <p>
          Illarin keeps the file you uploaded exactly as you uploaded it.
          Exports are generated from it; they never replace it.
        </p>

        <h3>4.3 What you are promising</h3>
        <p>When you upload something, you are telling us that:</p>
        <ul>
          <li>
            You own it, or you have the rights to upload it and grant the
            licence above.
          </li>
          <li>
            It does not infringe anyone else&rsquo;s copyright, trademark, or
            other rights.
          </li>
          <li>
            It follows the{" "}
            <Link href="/legal/acceptable-use">Acceptable Use Policy</Link>.
          </li>
        </ul>
      </>
    ),
    heading: "4. Your work",
  },
  {
    body: (
      <>
        <p>
          A published work is either listed or unlisted. Listed work appears in
          Browse, in search, and in the sitemap. Unlisted work does not, but
          anyone holding the address can still open it, download it, and see its
          images.
        </p>
        <p>
          <strong>Unlisted means harder to find, not private.</strong> It is a
          way of keeping something out of Browse, and it is not a security
          boundary. Do not use it to protect anything that would harm you if a
          stranger read it.
        </p>
        <p>
          Before you can publish, you have to answer whether the work contains
          adult content. Answering it wrongly is a violation of the{" "}
          <Link href="/legal/acceptable-use">Acceptable Use Policy</Link>.
        </p>
      </>
    ),
    heading: "5. Publishing and who can see your work",
  },
  {
    body: (
      <>
        <p>
          Moderation on Illarin is a person reading a report. There are no
          automated content classifiers, and nothing you upload is scanned or
          scored by a machine learning model.
        </p>
        <p>
          We may withhold a work, which takes it out of Browse and makes it
          answer as missing to everyone except you. While a work is withheld you
          can still read and download it, and you can see the reason it was
          withheld, but you cannot edit or delete it. We may also remove work
          outright and close accounts. We try to be fair, and we do not promise
          a formal appeal for every decision.
        </p>
      </>
    ),
    heading: "6. Moderation",
  },
  {
    body: (
      <>
        <p>
          Deleting a work hides it immediately and starts a 30 day recovery
          window during which you can restore it. After that window it is
          destroyed for good, and the file behind it is destroyed with it unless
          another work shares the same bytes.
        </p>
        <p>
          You can close your account at any time. Your handle is retired rather
          than freed, so nobody can take it afterwards.
        </p>
      </>
    ),
    heading: "7. Deleting your work and your account",
  },
  {
    body: (
      <>
        <p>
          You can connect an app to your account so that it can fetch your
          library. When you connect one you choose what it may do, and Illarin
          records the app&rsquo;s name, the permissions you granted, and when it
          last used them. You can revoke a connected app at any time, and
          revoking it immediately stops its access.
        </p>
      </>
    ),
    heading: "8. Apps you connect",
  },
  {
    body: (
      <>
        <p>
          The <Link href="/legal/acceptable-use">Acceptable Use Policy</Link> is
          part of these Terms. Breaking it can mean your work is removed, your
          account is suspended, or your account is closed.
        </p>
      </>
    ),
    heading: "9. Acceptable use",
  },
  {
    body: (
      <>
        <p>
          Illarin runs on a hosting provider&rsquo;s servers, sends email
          through an email provider, monitors its own logs through a monitoring
          provider, and offers Discord as a way to sign in. Those services have
          their own terms, and we are not responsible for how they behave. The{" "}
          <Link href="/legal/privacy">Privacy Policy</Link> says what each of
          them receives.
        </p>
      </>
    ),
    heading: "10. Services we depend on",
  },
  {
    body: (
      <>
        <p>
          The Illarin name, artwork, and design belong to us or to the people
          who licensed them to us. The source code is available under the
          licence in its repository. Nothing in these Terms takes away rights
          that licence grants you.
        </p>
      </>
    ),
    heading: "11. Illarin itself",
  },
  {
    body: (
      <>
        <p>
          If you believe something on Illarin infringes your copyright, the{" "}
          <Link href="/legal/dmca">DMCA and copyright policy</Link> explains how
          to tell us and what happens next.
        </p>
      </>
    ),
    heading: "12. Copyright",
  },
  {
    body: (
      <>
        <p>
          You can stop using Illarin and delete your account whenever you like.
          We may suspend or end your access, with or without notice, if we
          believe you have broken these Terms, or if Illarin shuts down. The
          parts of these Terms that should outlast the account — ownership, the
          licence covering copies that already exist, the disclaimers, the
          liability limit, and the dispute terms — carry on afterwards.
        </p>
      </>
    ),
    heading: "13. Ending things",
  },
  {
    body: (
      <>
        <p>
          Illarin is provided <strong>as is and as available</strong>, with no
          warranties of any kind, express or implied, including merchantability,
          fitness for a particular purpose, and non-infringement. It is one
          person&rsquo;s project. We do not promise it will stay up, stay fast,
          stay free of bugs, or keep running at all, and we do not promise that
          what other people publish on it is accurate, safe, or to your taste.
        </p>
        <p>
          Keep your own copies of work that matters to you. Illarin is not a
          backup service and does not promise that lost work can be recovered.
        </p>
      </>
    ),
    heading: "14. No warranty",
  },
  {
    body: (
      <>
        <p>
          As far as the law allows, we are not liable for indirect, incidental,
          special, consequential, or punitive damages, or for lost profits,
          revenue, data, or goodwill, arising from your use of Illarin. Our
          total liability for any claim will not exceed{" "}
          <strong>one hundred United States dollars</strong>.
        </p>
      </>
    ),
    heading: "15. Limit of liability",
  },
  {
    body: (
      <>
        <p>
          If someone brings a claim against us because of what you uploaded, how
          you used Illarin, or how you broke these Terms, you agree to cover the
          resulting claims, damages, liabilities, and reasonable legal costs.
        </p>
      </>
    ),
    heading: "16. Covering our costs if you cause them",
  },
  {
    body: (
      <>
        <p>
          These Terms are governed by the laws of the Commonwealth of Virginia,
          United States, without regard to its conflict-of-laws rules. Any
          dispute goes to the state or federal courts sitting in Virginia, and
          you agree those courts may hear it. None of this takes away rights you
          hold under the law of the country you live in that cannot be signed
          away.
        </p>
      </>
    ),
    heading: "17. Governing law",
  },
  {
    body: (
      <>
        <p>
          We may update these Terms. If a change matters, we will update the
          effective date at the top and, where it is reasonable to do so, say
          something through Illarin itself. Carrying on using Illarin after a
          change takes effect means you accept it.
        </p>
      </>
    ),
    heading: "18. Changes",
  },
  {
    body: (
      <>
        <p>
          Questions about these Terms go to{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
        </p>
      </>
    ),
    heading: "19. Contact",
  },
];
export default function Terms() {
  return (
    <LegalPage
      clauses={CLAUSES}
      href="/legal/terms"
      title="Terms of Service"
      lede={
        <>
          These Terms govern your use of Illarin at illarin.com. By creating an
          account, uploading work, or otherwise using Illarin, you agree to
          them. If you don&rsquo;t agree, don&rsquo;t use Illarin.
        </>
      }
    />
  );
}
