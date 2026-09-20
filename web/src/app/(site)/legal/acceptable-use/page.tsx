import Link from "next/link";
import { CONTACT_EMAIL } from "@/lib/contact";
import { pageMetadata } from "@/lib/site-metadata";
import { type LegalClause, LegalPage } from "../LegalPage";

export const metadata = pageMetadata(
  "Acceptable Use",
  "What you can publish on Illarin, what you cannot, and how that is enforced.",
);

const CLAUSES: LegalClause[] = [
  {
    body: (
      <>
        <p>
          None of the following belongs on Illarin, in any work, in any image,
          in a profile, or anywhere else:
        </p>
        <ul>
          <li>
            <strong>Sexual content involving minors</strong>, real or fictional.
            Any sexual or sexualised depiction of someone presented or implied
            to be under 18 is banned, however the character is framed. Calling a
            character ageless, ancient, or an adult-coded archetype does not get
            around this rule.
          </li>
          <li>
            <strong>
              Sexual content about real, identifiable people who have not
              consented
            </strong>
            , including deepfake-style material.
          </li>
          <li>
            <strong>
              Real-world violence aimed at identifiable people or groups
            </strong>
            : incitement, glorification, plans for an attack, or anything
            encouraging self-harm or suicide.
          </li>
          <li>
            <strong>Private personal information</strong> published without the
            person&rsquo;s consent, such as real names, home or work addresses,
            phone numbers, government identifiers, or financial details.
          </li>
          <li>
            <strong>Anything illegal</strong> where Illarin is hosted or where
            you are, including child sexual abuse material, terrorist material,
            and extreme gore.
          </li>
          <li>
            <strong>Instructions for serious real-world harm</strong>, including
            workable instructions for weapons capable of mass casualties,
            malware aimed at real systems, or attacks on infrastructure.
          </li>
        </ul>
      </>
    ),
    heading: "1. Never allowed",
  },
  {
    body: (
      <>
        <p>
          Adult work is welcome on Illarin as long as it stays inside the rules
          above and is marked correctly.
        </p>
        <ul>
          <li>
            Adult content is <strong>hidden by default</strong>. A reader has to
            choose to see it, and confirm they are 18 or older.
          </li>
          <li>
            Every work has to answer the adult content question before it can be
            published, and answering it wrongly is a violation on its own. If
            you are unsure, mark it as adult.
          </li>
          <li>
            Every sexual character has to be unmistakably 18 or older, in the
            writing and in any image.
          </li>
          <li>
            You can keep your adult work off your public profile with a setting,
            and that setting does not change any of the rules above.
          </li>
        </ul>
      </>
    ),
    heading: "2. Adult content",
  },
  {
    body: (
      <>
        <ul>
          <li>
            No targeted harassment, threats, or stalking, of people here or
            anywhere else.
          </li>
          <li>
            Nothing that dehumanises people or incites hatred against them based
            on race, ethnicity, national origin, religion, disability, gender,
            gender identity, sexual orientation, or anything else of that kind.
          </li>
          <li>
            Hard criticism, satire, and plain disagreement are fine. Personal
            attacks and slurs used as insults are not.
          </li>
        </ul>
      </>
    ),
    heading: "3. Harassment and hate",
  },
  {
    body: (
      <>
        <ul>
          <li>
            Don&rsquo;t upload work that infringes someone&rsquo;s copyright,
            trademark, or other rights.
          </li>
          <li>
            Fanwork is welcome when it follows the rest of this policy.
            Don&rsquo;t claim work that isn&rsquo;t yours as your own.
          </li>
          <li>
            If you are the rights holder, the{" "}
            <Link href="/legal/dmca">DMCA and copyright policy</Link> explains
            how to get something taken down.
          </li>
        </ul>
      </>
    ),
    heading: "4. Other people’s work",
  },
  {
    body: (
      <>
        <ul>
          <li>
            No spam, bulk uploads of the same thing, link farming, or affiliate
            bait.
          </li>
          <li>
            No inflating your own download counts or manipulating search
            results.
          </li>
          <li>
            No scraping or crawling beyond what Illarin&rsquo;s robots.txt and
            rate limits allow.
          </li>
          <li>
            No probing, scanning, or attacking Illarin, and no trying to get
            into anyone else&rsquo;s account.
          </li>
          <li>
            No malware, phishing, or misleading links in uploads or profiles.
          </li>
          <li>
            Don&rsquo;t use a connected app, or a token from one, to reach work
            you would not be allowed to reach in a browser.
          </li>
        </ul>
      </>
    ),
    heading: "5. Spam and abusing the platform",
  },
  {
    body: (
      <>
        <ul>
          <li>
            Don&rsquo;t pretend to be someone else, including other creators
            here, public figures, or us.
          </li>
          <li>Don&rsquo;t make a new account to get around a suspension.</li>
        </ul>
      </>
    ),
    heading: "6. Being straight about who you are",
  },
  {
    body: (
      <>
        <p>
          Moderation on Illarin is a person reading a report. There are no
          automated classifiers, and nothing you upload is scored by a machine.
        </p>
        <p>
          We may take down a work, which takes it out of Browse and makes it
          answer as missing to everyone but its creator, who can still read it,
          download it, and see why. We may also remove work outright, suspend an
          account, or close one. What happens depends on how serious it is and
          what came before. Anything in section 1 usually means immediate
          removal and can mean an immediate permanent ban.
        </p>
      </>
    ),
    heading: "7. How this is enforced",
  },
  {
    body: (
      <>
        <p>
          Write to <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
          Include the address of the work and what is wrong with it. For
          anything urgent — a credible threat, private information published
          about someone, or content covered by section 1 — say so in the subject
          line.
        </p>
      </>
    ),
    heading: "8. Reporting something",
  },
  {
    body: (
      <>
        <p>
          This policy changes as Illarin does. The effective date at the top
          says when the current wording took effect.
        </p>
      </>
    ),
    heading: "9. Changes",
  },
];
export default function AcceptableUse() {
  return (
    <LegalPage
      clauses={CLAUSES}
      href="/legal/acceptable-use"
      title="Acceptable Use Policy"
      lede={
        <>
          Illarin holds characters, lorebooks, presets, themes, and packs made
          by the people who use it. These rules apply to all of it, and to how
          you behave here. Breaking them can mean your work is taken down or
          removed, or your account is suspended or closed.
        </>
      }
    />
  );
}
