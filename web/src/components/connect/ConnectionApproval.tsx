"use client";

import { Check, CircleX, ShieldAlert, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  type FormEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { refusalMessage } from "@/lib/answer";
import { api } from "@/lib/api/client";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import {
  isLinkRedirect,
  isPendingDeviceLink,
  isPendingLink,
  isSafeLoopbackRedirect,
  type PendingLink,
} from "@/lib/link-request";
import { type Decision, LinkDecision } from "./LinkDecision";

type ReviewRequest =
  | { kind: "authorization"; requestCode: string }
  | { kind: "device"; userCode: string };

type ReviewSource =
  | { kind: "authorization"; requestCode: string }
  | { kind: "device"; userCode: string; approvalToken: string };

type Review = { source: ReviewSource; link: PendingLink };

type Stage =
  | { kind: "entry" }
  | { kind: "loading" }
  | { kind: "request-error" }
  | { kind: "confirm"; review: Review }
  | { kind: "deciding"; review: Review; decision: Decision }
  | { kind: "redirecting"; review: Review; decision: Decision }
  | { kind: "approved"; link: PendingLink }
  | { kind: "denied"; link: PendingLink };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function LinkApproval() {
  const search = useSearchParams();
  const { account } = useAuth();
  const requestCode = search.get("request")?.trim() ?? "";
  const returnTo = requestCode
    ? `/link?request=${encodeURIComponent(requestCode)}`
    : "/link";
  const [manualForRequest, setManualForRequest] = useState("");
  const [stage, setStage] = useState<Stage>(() =>
    requestCode ? { kind: "loading" } : { kind: "entry" },
  );
  const [typed, setTyped] = useState("");
  const [trouble, setTrouble] = useState("");
  const looked = useRef("");
  const activePanel = useRef<HTMLElement | null>(null);
  const previousView = useRef("");
  const reviewingRequest =
    requestCode !== "" && manualForRequest !== requestCode;
  const stageView =
    stage.kind === "deciding" || stage.kind === "redirecting"
      ? "confirm"
      : stage.kind;
  const viewKey =
    account === undefined
      ? "checking-account"
      : !account
        ? "signed-out"
        : !account.emailVerified
          ? "unverified"
          : stageView;
  const capturePanel = useCallback((node: HTMLElement | null) => {
    activePanel.current = node;
  }, []);

  useEffect(() => {
    if (previousView.current && previousView.current !== viewKey) {
      activePanel.current?.focus();
    }
    previousView.current = viewKey;
  }, [viewKey]);

  const loadReview = useCallback(async (request: ReviewRequest) => {
    setTrouble("");
    setStage({ kind: "loading" });

    const isAuthorization = request.kind === "authorization";
    const endpoint = isAuthorization
      ? `/v1/link/authorizations/${encodeURIComponent(request.requestCode)}`
      : `/v1/link/requests/${encodeURIComponent(request.userCode)}`;

    try {
      const { data, error, response } = await api<unknown>("GET", endpoint, {
        cache: "no-store",
      });
      const answer = response.ok ? data : error;
      if (!response.ok) {
        setTrouble(
          refusalMessage(
            answer,
            isAuthorization
              ? "That browser link request is no longer available."
              : "That code does not match a pending link request.",
          ),
        );
        setStage({ kind: isAuthorization ? "request-error" : "entry" });
        return;
      }

      if (isAuthorization) {
        if (!isPendingLink(answer)) {
          setTrouble("Illarin returned an incomplete link request. Try again.");
          setStage({ kind: "request-error" });
          return;
        }
        setStage({
          kind: "confirm",
          review: {
            link: answer,
            source: { kind: "authorization", requestCode: request.requestCode },
          },
        });
        return;
      }

      if (!isPendingDeviceLink(answer)) {
        setTrouble("Illarin returned an incomplete link request. Try again.");
        setStage({ kind: "entry" });
        return;
      }
      setStage({
        kind: "confirm",
        review: {
          link: answer,
          source: {
            approvalToken: answer.approvalToken,
            kind: "device",
            userCode: request.userCode,
          },
        },
      });
    } catch {
      setTrouble(UNREACHABLE);
      setStage({ kind: isAuthorization ? "request-error" : "entry" });
    }
  }, []);

  useEffect(() => {
    if (!reviewingRequest || !account?.emailVerified) return;
    if (looked.current === requestCode) return;
    looked.current = requestCode;
    void loadReview({ kind: "authorization", requestCode });
  }, [requestCode, reviewingRequest, account, loadReview]);

  function startOver() {
    setTrouble("");
    setTyped("");
    setStage({ kind: "entry" });
  }

  if (account === undefined) {
    return (
      <Frame lede="">
        <Panel capture={capturePanel}>
          <p className="font-ui text-ui text-mute">
            Checking who is signed in…
          </p>
        </Panel>
      </Frame>
    );
  }

  if (!account) {
    return (
      <Frame lede="">
        <Panel capture={capturePanel}>
          <Gate
            action={
              <Button asChild size="large" variant="primary">
                <Link
                  href={`/sign-in?returnTo=${encodeURIComponent(returnTo)}`}
                >
                  Sign in
                </Link>
              </Button>
            }
            body="Sign in to the account you want to link to this application."
            requestPending={reviewingRequest}
            title="Sign in to review this link"
          />
        </Panel>
      </Frame>
    );
  }

  if (!account.emailVerified) {
    return (
      <Frame lede="">
        <Panel capture={capturePanel}>
          <Gate
            action={
              <Button asChild size="large" variant="primary">
                <Link
                  href={`/verify-email?returnTo=${encodeURIComponent(returnTo)}`}
                >
                  Verify email
                </Link>
              </Button>
            }
            body="A verified address is needed before an application can be linked to your account."
            requestPending={reviewingRequest}
            title="Verify your email first"
          />
        </Panel>
      </Frame>
    );
  }

  if (stage.kind === "approved") {
    return (
      <Frame lede="">
        <Panel capture={capturePanel}>
          <Landing
            action={
              <Button asChild size="large" variant="primary">
                <Link href="/settings">See linked applications</Link>
              </Button>
            }
            body={`Go back to ${stage.link.applicationName}. Linking is approved. The application can now finish connecting.`}
            mark={
              <Mark tone="accent">
                <Check aria-hidden="true" className="size-6" strokeWidth={2} />
              </Mark>
            }
            title={`${stage.link.instanceName} is linked`}
          />
        </Panel>
      </Frame>
    );
  }

  if (stage.kind === "denied") {
    return (
      <Frame lede="">
        <Panel capture={capturePanel}>
          <Landing
            action={
              <Button onClick={startOver} size="large" variant="secondary">
                Enter another code
              </Button>
            }
            body={`${stage.link.applicationName} was not linked. The application will see that this request was denied.`}
            mark={
              <Mark tone="stop">
                <CircleX
                  aria-hidden="true"
                  className="size-6"
                  strokeWidth={1.6}
                />
              </Mark>
            }
            title="Link declined"
          />
        </Panel>
      </Frame>
    );
  }

  if (
    stage.kind === "confirm" ||
    stage.kind === "deciding" ||
    stage.kind === "redirecting"
  ) {
    const review = stage.review;
    const isDevice = review.source.kind === "device";
    return (
      <Frame
        lede={
          isDevice
            ? "Approve a private connection between your Illarin account and the installation that showed you this code."
            : "Approve a private connection between your Illarin account and the application that opened this page."
        }
      >
        <Panel capture={capturePanel}>
          <LinkDecision
            deciding={stage.kind === "confirm" ? null : stage.decision}
            link={review.link}
            onCancel={isDevice ? startOver : undefined}
            onDecide={(decision) => {
              void decide(review, decision, setStage, setTrouble);
            }}
            trouble={trouble}
            userCode={
              review.source.kind === "device"
                ? review.source.userCode
                : undefined
            }
          />
        </Panel>
      </Frame>
    );
  }

  if (stage.kind === "loading") {
    return (
      <Frame lede="">
        <Panel busy capture={capturePanel}>
          <p className="font-ui text-ui text-mute">Loading the link request…</p>
        </Panel>
      </Frame>
    );
  }

  if (stage.kind === "request-error" && reviewingRequest) {
    return (
      <Frame lede="The application could not reopen its request. You can enter a device code instead.">
        <Panel capture={capturePanel}>
          <Mark tone="stop">
            <ShieldAlert
              aria-hidden="true"
              className="size-6"
              strokeWidth={1.6}
            />
          </Mark>
          <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
            This request could not be opened
          </h2>
          <p className="mt-3 max-w-[52ch] font-prose text-prose text-mute">
            It may have expired or already been used. Reopen the link from your
            application, or enter a device code instead.
          </p>
          {trouble ? (
            <div className="mt-5 max-w-[34rem]">
              <Trouble>{trouble}</Trouble>
            </div>
          ) : null}
          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Button
              onClick={() => {
                setManualForRequest(requestCode);
                setTrouble("");
                setStage({ kind: "entry" });
              }}
              size="large"
              variant="primary"
            >
              Enter a device code
            </Button>
            <Button
              onClick={() => {
                looked.current = requestCode;
                void loadReview({ kind: "authorization", requestCode });
              }}
              variant="outline"
            >
              Try again
            </Button>
          </div>
        </Panel>
      </Frame>
    );
  }

  return (
    <Frame lede="Start linking in your application. If it gives you a code, enter it here.">
      <form
        noValidate
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          const userCode = typed.trim();
          if (!userCode) {
            setTrouble("Enter the code shown by your application.");
            return;
          }
          void loadReview({ kind: "device", userCode });
        }}
        ref={capturePanel}
        className="outline-none focus-visible:outline-none"
        tabIndex={-1}
      >
        <label
          className="font-display text-section font-medium tracking-tight text-ink"
          htmlFor="link-code"
        >
          Type the code your application is showing
        </label>
        <input
          aria-describedby={
            trouble ? "link-entry-trouble" : "link-entry-reason"
          }
          autoCapitalize="characters"
          autoComplete="off"
          className="mt-5 block min-h-[4.5rem] w-full max-w-[26rem] rounded-plate border-0 bg-deep px-6 text-center font-mono text-[clamp(1.6rem,4.5vw,2.5rem)] tracking-[0.22em] text-ink uppercase outline-offset-2 placeholder:text-mute/45"
          enterKeyHint="go"
          id="link-code"
          maxLength={12}
          name="code"
          onChange={(event) => setTyped(event.target.value.toUpperCase())}
          placeholder="XXXX-XXXX"
          required
          spellCheck={false}
          value={typed}
        />
        {trouble ? (
          <div className="mt-4" id="link-entry-trouble">
            <Trouble>{trouble}</Trouble>
          </div>
        ) : null}
        <div className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-4">
          <Button size="large" type="submit" variant="primary">
            Review request
          </Button>
          <p
            className="min-w-0 max-w-[34ch] flex-1 font-prose text-meta text-mute"
            id="link-entry-reason"
          >
            Next, review the application's identity and requested permissions.
          </p>
        </div>
      </form>
    </Frame>
  );
}

function Frame({ children, lede }: { children: ReactNode; lede: string }) {
  return (
    <>
      <header>
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          Link an application
        </h1>
        {lede ? (
          <p className="mt-4 max-w-[58ch] font-prose text-lede text-mute">
            {lede}
          </p>
        ) : null}
      </header>
      <div className="mt-9">{children}</div>
    </>
  );
}

function Panel({
  busy,
  capture,
  children,
}: {
  busy?: boolean;
  capture: (node: HTMLElement | null) => void;
  children: ReactNode;
}) {
  return (
    <div
      aria-busy={busy}
      aria-live="polite"
      ref={capture}
      className="outline-none focus-visible:outline-none"
      tabIndex={-1}
    >
      {children}
    </div>
  );
}

function Mark({
  children,
  tone,
}: {
  children: ReactNode;
  tone: "accent" | "stop";
}) {
  return (
    <span
      className={cn(
        "grid size-12 place-items-center rounded-plate",
        tone === "accent"
          ? "bg-accent-wash text-accent"
          : "bg-stop-wash text-stop",
      )}
    >
      {children}
    </span>
  );
}

function Landing({
  action,
  body,
  mark,
  title,
}: {
  action: ReactNode;
  body: string;
  mark: ReactNode;
  title: string;
}) {
  return (
    <div className="max-w-[34rem]">
      {mark}
      <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink [overflow-wrap:anywhere]">
        {title}
      </h2>
      <p className="mt-3 font-prose text-prose text-mute">{body}</p>
      <div className="mt-7">{action}</div>
    </div>
  );
}

function Gate({
  action,
  body,
  requestPending,
  title,
}: {
  action: ReactNode;
  body: string;
  requestPending: boolean;
  title: string;
}) {
  return (
    <div className="max-w-[34rem]">
      <Mark tone="accent">
        <ShieldCheck aria-hidden="true" className="size-6" strokeWidth={1.5} />
      </Mark>
      <h2 className="mt-5 font-display text-title font-medium tracking-tight text-ink">
        {title}
      </h2>
      <p className="mt-3 font-prose text-prose text-mute">{body}</p>
      {requestPending ? (
        <p className="mt-3 font-ui text-meta text-mute">
          The browser request will remain available after this account step.
        </p>
      ) : null}
      <div className="mt-7">{action}</div>
    </div>
  );
}

async function decide(
  review: Review,
  decision: Decision,
  setStage: (stage: Stage) => void,
  setTrouble: (trouble: string) => void,
) {
  setTrouble("");
  setStage({ decision, kind: "deciding", review });
  const { source } = review;
  const isAuthorization = source.kind === "authorization";
  const action = decision === "approve" ? "approve" : "deny";
  const endpoint = isAuthorization
    ? `/v1/link/authorizations/${encodeURIComponent(source.requestCode)}/${action}`
    : `/v1/link/requests/${encodeURIComponent(source.userCode)}/${action}`;
  const body = isAuthorization
    ? undefined
    : { approvalToken: source.approvalToken };

  try {
    const { data, error, response } = await api<unknown>("POST", endpoint, {
      body,
    });
    const answer = response.ok ? data : error;
    if (!response.ok) {
      setTrouble(
        refusalMessage(
          answer,
          decision === "approve"
            ? "This link request could not be approved."
            : "This link request could not be declined.",
        ),
      );
      setStage({ kind: "confirm", review });
      return;
    }

    if (isAuthorization) {
      if (
        !isLinkRedirect(answer) ||
        !isSafeLoopbackRedirect(answer.redirectUrl)
      ) {
        setTrouble(
          "Illarin returned an unsafe callback address. Nothing was opened.",
        );
        setStage({ kind: "confirm", review });
        return;
      }
      setStage({ decision, kind: "redirecting", review });
      window.location.assign(answer.redirectUrl);
      return;
    }

    if (decision === "deny") {
      setStage({ kind: "denied", link: review.link });
      return;
    }

    if (!isPendingLink(answer)) {
      setTrouble("Illarin returned an incomplete approval. Try again.");
      setStage({ kind: "confirm", review });
      return;
    }
    setStage({ kind: "approved", link: answer });
  } catch {
    setTrouble(UNREACHABLE);
    setStage({ kind: "confirm", review });
  }
}
