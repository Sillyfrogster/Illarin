import Link from "next/link";
import { CONTACT_EMAIL, KOFI_PAGE } from "@/lib/contact";
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
          behind it. It charges nothing today. It carries no ads, and it never
          will. &ldquo;We,&rdquo; &ldquo;us,&rdquo; and &ldquo;Illarin&rdquo;
          mean the person who runs it, reachable at{" "}
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
          You keep ownership of the characters, lorebooks, presets, themes,
          packs, and extensions you upload. You are responsible for them.
        </p>

        <h3>4.2 What you let us do with it</h3>
        <p>
          So that Illarin can do its job, you grant us a worldwide,
          non-exclusive, royalty-free license to store your work, show it to the
          people you publish it to, hand it to the apps you or your readers
          connect, resize your images for display, and convert your work into
          formats for other apps. That license covers nothing else, and it ends
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
            license above.
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
          We may take down a work, which takes it out of Browse and makes it
          answer as missing to everyone except you. While a work is taken down
          you can still read and download it, and you can see the reason it was
          taken down, but you cannot edit or delete it. We may also remove work
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
          Illarin charges nothing today. You can support it on{" "}
          <a href={KOFI_PAGE}>Ko-fi</a>. That is a gift to the person who runs
          Illarin: it buys nothing here, and Ko-fi&rsquo;s terms cover the
          payment.
        </p>
        <p>
          Illarin may later add the features below. Each part applies once its
          feature exists. Before you pay, the page you pay on shows what you pay
          and what Illarin keeps.
        </p>

        <h3>9.1 Payments go through a provider</h3>
        <p>
          Every payment on Illarin goes through a payment provider. Illarin
          never holds your card or bank details, and the provider&rsquo;s terms
          apply alongside these.
        </p>

        <h3>9.2 Donations</h3>
        <p>
          You can give money to Illarin or to a creator. A donation is a gift.
          It buys nothing, and it is not refunded unless the law or the provider
          requires it.
        </p>

        <h3>9.3 Membership</h3>
        <p>
          A membership is a payment to Illarin that repeats until you cancel it.
          Canceling stops the next payment, and the membership lasts until the
          end of the period you already paid for.
        </p>

        <h3>9.4 Commissions</h3>
        <p>
          A creator can take paid commissions through Illarin. A commission is
          an agreement between you and the creator. The creator is responsible
          for delivering what they agreed to, and Illarin is not a party to that
          agreement. Illarin keeps a cut of each commission, shown before you
          pay. If a commission is not delivered, Illarin may refund it through
          the provider.
        </p>

        <h3>9.5 Paying creators</h3>
        <p>
          Money a creator earns is paid out through the provider, to the account
          the creator sets up there. The provider may need the creator&rsquo;s
          identity and tax details before it pays, and the creator is
          responsible for their own taxes.
        </p>

        <h3>9.6 Money when an account closes</h3>
        <ul>
          <li>
            Any membership is canceled, so no further payment is taken. The part
            of a period already paid is not refunded.
          </li>
          <li>
            Commissions not yet delivered are refunded to the person who paid.
          </li>
          <li>
            Money a creator has already earned is still paid out through the
            provider, as long as the provider can pay it.
          </li>
          <li>
            If we close an account for breaking these Terms or the{" "}
            <Link href="/legal/acceptable-use">Acceptable Use Policy</Link>,
            money tied to the breach may be held back or returned to the people
            who paid it, where the law and the provider allow.
          </li>
          <li>
            Payment records are kept for as long as tax law requires, even after
            the account is gone.
          </li>
        </ul>
      </>
    ),
    heading: "9. Money",
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
    heading: "10. Acceptable use",
  },
  {
    body: (
      <>
        <p>
          Illarin runs on a hosting provider&rsquo;s servers, sends email
          through an email provider, sends its server logs to Datadog for
          monitoring, and offers Discord as a way to sign in. Support goes
          through Ko-fi, and payments, once Illarin takes them, go through a
          payment provider. Those services have their own terms, and we are not
          responsible for how they behave. The{" "}
          <Link href="/legal/privacy">Privacy Policy</Link> says what each of
          them receives.
        </p>
      </>
    ),
    heading: "11. Services we depend on",
  },
  {
    body: (
      <>
        <p>
          The Illarin name, artwork, and design belong to us or to the people
          who licensed them to us. The source code is available under the
          license in its repository. Nothing in these Terms takes away rights
          that license grants you.
        </p>
      </>
    ),
    heading: "12. Illarin itself",
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
    heading: "13. Copyright",
  },
  {
    body: (
      <>
        <p>
          You can stop using Illarin and delete your account whenever you like.
          We may suspend or end your access, with or without notice, if we
          believe you have broken these Terms, or if Illarin shuts down. The
          parts of these Terms that should outlast the account — ownership, the
          license covering copies that already exist, the disclaimers, the
          liability limit, and the dispute terms — carry on afterwards.
        </p>
      </>
    ),
    heading: "14. Ending things",
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
    heading: "15. No warranty",
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
    heading: "16. Limit of liability",
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
    heading: "17. Covering our costs if you cause them",
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
    heading: "18. Governing law",
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
    heading: "19. Changes",
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
    heading: "20. Contact",
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
