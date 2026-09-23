import { CONTACT_EMAIL } from "@/lib/contact";
import { pageMetadata } from "@/lib/site-metadata";
import { type LegalClause, LegalPage } from "../LegalPage";

export const metadata = pageMetadata(
  "DMCA / Copyright",
  "How to report copyright infringement on Illarin and what happens after you do.",
);

const CLAUSES: LegalClause[] = [
  {
    body: (
      <>
        <p>
          Email <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>. Email
          is the only channel, and it is read by the person who runs Illarin.
        </p>
        <p>
          <strong>One thing to know first:</strong> Illarin is a personal
          project and has not registered a designated agent with the United
          States Copyright Office, so it does not claim the DMCA safe harbor
          that registration provides. We follow the process below anyway,
          because it is the right way to handle a copyright complaint and it
          gives both sides a fair hearing.
        </p>
      </>
    ),
    heading: "1. Where to send a notice",
  },
  {
    body: (
      <>
        <p>Include all of this, or we may not be able to act:</p>
        <ol>
          <li>
            Your signature, physical or electronic, as the copyright owner or
            someone authorized to act for them.
          </li>
          <li>
            What work you say has been infringed. If it is many works at once, a
            representative list is enough.
          </li>
          <li>
            What on Illarin you say is infringing, precisely enough for us to
            find it. The full address of the page is best.
          </li>
          <li>Your name, address, telephone number, and email address.</li>
          <li>
            A statement that you believe in good faith that the use complained
            of is not authorized by the copyright owner, its agent, or the law.
          </li>
          <li>
            A statement, under penalty of perjury, that what you have said is
            accurate and that you are the copyright owner or authorized to act
            for them.
          </li>
        </ol>
      </>
    ),
    heading: "2. What a notice needs to say",
  },
  {
    body: (
      <>
        <ul>
          <li>
            If the notice is complete and the claim is clear, we take the
            material down or cut off access to it within a reasonable time.
          </li>
          <li>
            We make a reasonable effort to tell the creator who uploaded it, and
            to pass on a copy of the notice.
          </li>
          <li>Accounts that keep infringing are closed.</li>
        </ul>
      </>
    ),
    heading: "3. What happens next",
  },
  {
    body: (
      <>
        <p>Send a counter-notice to the same address, containing:</p>
        <ol>
          <li>Your signature, physical or electronic.</li>
          <li>What was removed, and the address where it used to be.</li>
          <li>
            A statement, under penalty of perjury, that you believe in good
            faith it was removed through a mistake or a misidentification.
          </li>
          <li>
            Your name, address, and telephone number; a statement that you
            accept the jurisdiction of the federal district court for the
            district you live in, or, if you live outside the United States, the
            United States District Court for the Eastern District of Virginia;
            and a statement that you will accept service of process from whoever
            sent the original notice.
          </li>
        </ol>
        <p>
          If the counter-notice is complete, we may pass it to whoever sent the
          original notice. Unless they tell us they have gone to court to stop
          you, we may restore what was removed after a reasonable wait.
        </p>
      </>
    ),
    heading: "4. If we took down your work by mistake",
  },
  {
    body: (
      <>
        <p>
          Knowingly lying in a notice or a counter-notice can make you liable
          under 17 U.S.C. § 512(f), including for the other side&rsquo;s damages
          and legal costs. Don&rsquo;t send a claim you don&rsquo;t mean.
        </p>
      </>
    ),
    heading: "5. False claims",
  },
  {
    body: (
      <>
        <p>
          For anything that is not copyright — a trademark, a right of
          publicity, an impersonation — write to{" "}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a> as well. Those
          are handled case by case rather than through the process above.
        </p>
      </>
    ),
    heading: "6. Trademark and other rights",
  },
];
export default function Dmca() {
  return (
    <LegalPage
      clauses={CLAUSES}
      href="/legal/dmca"
      title="DMCA / Copyright Policy"
      lede={
        <>
          Illarin acts on clear reports that something here infringes a
          copyright. This page says how to send one, what we do with it, and how
          to answer if your work was taken down by mistake.
        </>
      }
    />
  );
}
