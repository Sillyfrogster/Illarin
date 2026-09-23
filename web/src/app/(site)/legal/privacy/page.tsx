import Link from "next/link";
import { CONTACT_EMAIL, KOFI_PAGE } from "@/lib/contact";
import { pageMetadata } from "@/lib/site-metadata";
import { type LegalClause, LegalPage } from "../LegalPage";

export const metadata = pageMetadata(
  "Privacy Policy",
  "What Illarin collects, what it never collects, and who else sees any of it.",
);

const CLAUSES: LegalClause[] = [
  {
    body: (
      <>
        <h3>1.1 Your account</h3>
        <p>
          If you sign up with an email address, we store the address and a hash
          of your password. We never store the password itself. If you sign in
          with Discord, we store your Discord user ID, and we receive the
          username and avatar Discord exposes. We also receive the email address
          on your Discord account when Discord says it is verified, and we use
          it to fill in your Illarin address only if you have none.
        </p>
        <p>
          Every account also has the handle you chose, and the date it was
          created.
        </p>

        <h3>1.2 What you upload</h3>
        <p>
          The file you uploaded, kept exactly as you uploaded it. The name,
          blurb, tags, and adult content answer you attached. The images you
          added. Anything written inside the work itself.
        </p>

        <h3>1.3 Your settings</h3>
        <p>
          Whether you want to see adult content, and whether your adult work
          appears on your public profile.
        </p>

        <h3>1.4 Apps you connect</h3>
        <p>
          For each app you connect: the name and version it reported, the
          permissions you granted it, when you connected it, and when it last
          used its access. Its access tokens are stored only as hashes.
        </p>

        <h3>1.5 Server logs</h3>
        <p>
          Our web server records the usual line for each request: the IP address
          it came from, the time, the address requested, the response status,
          and the browser&rsquo;s user agent string. One-time link codes are
          stripped out of that line before it is written.
        </p>

        <h3>1.6 Page views</h3>
        <p>
          Illarin counts page views with Umami, an analytics tool we run on our
          own servers, so the counts never go to another company. For each page
          you open, it records the page address without anything after a
          &ldquo;?&rdquo; or &ldquo;#&rdquo;, the page title, the address of the
          page that linked you here, your browser, operating system, device
          type, screen size, and language, and the country, region, and city
          your IP address points to. It sets no cookie, stores nothing in your
          browser, and keeps neither your IP address nor anything tied to your
          account, so no identity is kept. Every page view record is deleted
          after 30 days. What stays is the number of visits on each day, and
          that number is kept.
        </p>

        <h3>1.7 Counts of what happens</h3>
        <p>
          When an account is created, a work is downloaded or sent to an app, or
          a work is published, Illarin records which of those happened, the work
          where there is one, and the day. The record holds no account, no IP
          address, no time of day, and nothing about the browser. Each night
          Illarin adds these records up into totals for each day and deletes the
          records older than 30 days. The daily totals are kept.
        </p>

        <h3>1.8 Monitoring</h3>
        <p>
          Illarin&rsquo;s servers send their logs to Datadog, a monitoring
          service, so that errors and outages show up. Those logs hold the
          request lines described in 1.5 and the errors Illarin&rsquo;s own
          programs write. Datadog also receives how busy the servers are and
          which programs are running on them. Nothing from Datadog runs in your
          browser.
        </p>

        <h3>1.9 Money</h3>
        <p>
          Illarin takes no payments today. If you support Illarin on{" "}
          <a href={KOFI_PAGE}>Ko-fi</a>, Ko-fi handles the payment under its own
          terms, and Illarin sees only what Ko-fi shows the owner of a page,
          such as the name and message you leave. Illarin does not tie it to
          your account.
        </p>
        <p>
          Illarin may later let you donate, take out a membership, or pay a
          creator for commissioned work. When it does, a payment provider
          handles the payment, and your card or bank details go to that
          provider, never to Illarin. Illarin keeps what it needs to run those
          features and meet tax law: who paid whom, how much, when, what for,
          and the provider&rsquo;s reference for the payment. A creator who is
          paid gives their payout details to the provider, not to Illarin. This
          section applies once those features exist.
        </p>
      </>
    ),
    heading: "1. What we collect",
  },
  {
    body: (
      <>
        <ul>
          <li>
            <strong>We do not record who downloads what.</strong> Illarin counts
            downloads for a creator&rsquo;s benefit. The record holds the work,
            the format, the time, and whether the download came from the
            creator, a connected app, or anyone else. It holds no account, no IP
            address, and nothing else that could point back to a person.
          </li>
          <li>
            <strong>
              There are no ads on Illarin, and there never will be.
            </strong>{" "}
            There are no advertising or cross-site tracking cookies, and no
            third-party script tracking what you read. Illarin does not sell
            your data or share it with advertisers.
          </li>
          <li>
            <strong>
              Nothing you upload is scanned by a machine learning model.
            </strong>{" "}
            Illarin runs no content classifiers, and it does not use your work
            as training data.
          </li>
        </ul>
      </>
    ),
    heading: "2. What we do not collect",
  },
  {
    body: (
      <>
        <p>
          Illarin sets a cookie holding your session when you sign in, and two
          short-lived cookies during a Discord sign-in so that it can bring you
          back to the page you started from. All three are strictly necessary to
          sign you in. Counting page views sets no cookie.
        </p>
        <p>
          Your browser also keeps a few display choices locally, which never
          reach us: light or dark appearance, whether the front page moves, how
          wide the side panel is, which follow suggestions you dismissed, and,
          for the length of the tab, how adult content is shown.
        </p>
      </>
    ),
    heading: "3. Cookies and what your browser stores",
  },
  {
    body: (
      <>
        <ul>
          <li>
            To run Illarin: signing you in, showing your work, search,
            downloads, and exports.
          </li>
          <li>
            To hand your library to the apps you have connected, within what you
            granted.
          </li>
          <li>
            To send you the emails the account needs, which are address
            verification and password resets.
          </li>
          <li>
            To act on reports and enforce the{" "}
            <Link href="/legal/acceptable-use">Acceptable Use Policy</Link>.
          </li>
          <li>
            To keep Illarin working and to see what broke when it does not.
          </li>
          <li>
            To see which pages people read, which sites send them here, and how
            many works are downloaded, sent, and published each day.
          </li>
          <li>
            To take and pass on payments, once Illarin offers them, and to keep
            the records tax law requires.
          </li>
        </ul>
      </>
    ),
    heading: "4. Why we use it",
  },
  {
    body: (
      <>
        <p>
          We rely on <strong>contract</strong> to give you the service you
          signed up for, <strong>legitimate interests</strong> to keep Illarin
          secure and to act on reports, <strong>consent</strong> where you have
          opted in, which is how adult content works, and{" "}
          <strong>legal obligation</strong> where the law requires us to
          respond.
        </p>
      </>
    ),
    heading: "5. Legal bases, if you are in the EEA or UK",
  },
  {
    body: (
      <>
        <p>
          We do not sell your personal data, and we have no interest in doing
          so. It reaches:
        </p>
        <ul>
          <li>
            <strong>Our hosting provider</strong>, whose servers hold the
            database, the uploaded files, and the logs.
          </li>
          <li>
            <strong>Our email provider</strong>, which receives your address and
            the message when Illarin sends a verification or password-reset
            email.
          </li>
          <li>
            <strong>Datadog</strong>, our monitoring service, which receives the
            server logs described above.
          </li>
          <li>
            <strong>Discord</strong>, if you sign in or link with it, or connect
            a channel for announcements. A connected channel receives the public
            page details of what you publish.
          </li>
          <li>
            <strong>Ko-fi</strong>, if you support Illarin there.
          </li>
          <li>
            <strong>A payment provider</strong>, once Illarin takes payments,
            which receives what it needs to charge you or pay you.
          </li>
          <li>
            <strong>Apps you connect</strong>, which receive the work they are
            allowed to fetch.
          </li>
          <li>
            <strong>Anyone</strong>, for work you publish and for your profile.
            That is the point of publishing.
          </li>
          <li>
            <strong>Authorities</strong>, where the law, a court order, or a
            genuine safety risk requires it.
          </li>
        </ul>
      </>
    ),
    heading: "6. Who else sees it",
  },
  {
    body: (
      <>
        <p>
          Account data lasts as long as your account. Work you publish lasts
          until you delete it or it is removed. Deleted work sits in a 30 day
          recovery window while you can still restore it, and is destroyed after
          that. Sign-in sessions, email verification links, password reset
          links, and app connection codes all expire on their own. Page view
          records and the counts described in 1.7 are deleted after 30 days, and
          the daily totals made from them are kept. Server logs are kept for a
          short rolling window. Once Illarin takes payments, payment records are
          kept for as long as tax law requires, even after the account is gone.
        </p>
      </>
    ),
    heading: "7. How long we keep it",
  },
  {
    body: (
      <>
        <p>
          You can see and change most of your data in your account settings,
          including your email address, your password, your handle, your
          connected apps, and your content preferences. Deleting a work or your
          account is a normal control, not a request you have to file.
        </p>
        <p>
          For anything else — a copy of your data, a correction you cannot make
          yourself, an objection, or withdrawing consent — write to{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>. If you are in
          California you have further rights under the CCPA and CPRA. If you are
          in the EEA or UK you have further rights under the GDPR and UK GDPR,
          including the right to complain to your data protection authority.
        </p>
      </>
    ),
    heading: "8. Your rights",
  },
  {
    body: (
      <>
        <p>
          Illarin is not for children under 13, and we do not knowingly collect
          anything from them. If you believe a child has given us personal data,
          write to <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a> and
          we will delete it.
        </p>
      </>
    ),
    heading: "9. Children",
  },
  {
    body: (
      <>
        <p>
          Illarin&rsquo;s servers, and the email, monitoring, sign-in, and
          payment services it depends on, operate across several countries
          including the United States. Using Illarin means your data travels to
          those places. Where the law requires safeguards for that transfer, we
          rely on the ones our providers put in place, such as Standard
          Contractual Clauses.
        </p>
      </>
    ),
    heading: "10. Where your data goes",
  },
  {
    body: (
      <>
        <p>
          Passwords are hashed, never stored in readable form. Session tokens,
          password reset links, email verification links, and app tokens are
          stored as hashes too, so a copy of the database does not hand someone
          your account. Traffic to Illarin is encrypted in transit.
        </p>
        <p>
          None of that makes a system perfectly secure. Use a password you use
          nowhere else, keep your Discord account secure, and write to{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a> if you think
          someone has got into your account.
        </p>
      </>
    ),
    heading: "11. Security",
  },
  {
    body: (
      <>
        <p>
          We may update this policy. The effective date at the top says when the
          current wording took effect, and for a change that matters we will say
          something through Illarin itself.
        </p>
      </>
    ),
    heading: "12. Changes",
  },
  {
    body: (
      <>
        <p>
          Privacy questions and requests go to{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
        </p>
      </>
    ),
    heading: "13. Contact",
  },
];
export default function Privacy() {
  return (
    <LegalPage
      clauses={CLAUSES}
      href="/legal/privacy"
      title="Privacy Policy"
      lede={
        <>
          This policy says what Illarin collects, why, who else sees it, and
          what you can do about it. Illarin is a personal project that shows no
          ads and has no reason to collect anything it does not need.
        </>
      }
    />
  );
}
