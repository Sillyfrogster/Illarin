"use client";

import { Check, CircleX, ShieldCheck } from "lucide-react";
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
  isConnectionRedirect,
  isPendingCodeConnection,
  isPendingConnection,
  isSafeLoopbackRedirect,
  type PendingConnection,
} from "@/lib/connection-request";
import { ConnectionDecision, type Decision } from "./ConnectionDecision";

type ReviewRequest =
  | { kind: "authorization"; requestCode: string; userCode: string }
  | { kind: "device"; userCode: string };

type ReviewSource = ReviewRequest & { approvalToken: string };

type Review = { source: ReviewSource; connection: PendingConnection };

type Stage =
  | { kind: "entry" }
  | { kind: "loading" }
  | { kind: "confirm"; review: Review }
  | { kind: "deciding"; review: Review; decision: Decision }
  | { kind: "redirecting"; review: Review; decision: Decision }
  | { kind: "approved"; connection: PendingConnection }
  | { kind: "denied"; connection: PendingConnection };

const UNREACHABLE =
  "We could not reach Illarin. Check your connection and try again.";

export function ConnectionApproval() {
  const search = useSearchParams();
  const { account } = useAuth();
  const requestCode = search.get("request")?.trim() ?? "";
  const returnTo = requestCode
    ? `/connect?request=${encodeURIComponent(requestCode)}`
    : "/connect";
  const [stage, setStage] = useState<Stage>({ kind: "entry" });
  const [typed, setTyped] = useState("");
  const [trouble, setTrouble] = useState("");
  const activePanel = useRef<HTMLElement | null>(null);
  const previousView = useRef("");
  const reviewingRequest = requestCode !== "";
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

    const endpoint =
      request.kind === "authorization"
        ? `/v1/connect/authorizations/${encodeURIComponent(request.requestCode)}?userCode=${encodeURIComponent(request.userCode)}`
        : `/v1/connect/requests/${encodeURIComponent(request.userCode)}`;

    try {
      const { data, error, response } = await api<unknown>("GET", endpoint, {
        cache: "no-store",
      });
      const answer = response.ok ? data : error;
      if (!response.ok) {
        setTrouble(
          refusalMessage(
            answer,
            "That code does not match a pending connection request.",
          ),
        );
        setStage({ kind: "entry" });
        return;
      }

      if (!isPendingCodeConnection(answer)) {
        setTrouble(
          "Illarin returned an incomplete connection request. Try again.",
        );
        setStage({ kind: "entry" });
        return;
      }
      setStage({
        kind: "confirm",
        review: {
          connection: answer,
          source: { ...request, approvalToken: answer.approvalToken },
        },
      });
    } catch {
      setTrouble(UNREACHABLE);
      setStage({ kind: "entry" });
    }
  }, []);

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
            body="Sign in to the account you want to connect this app to."
            requestPending={reviewingRequest}
            title="Sign in to review this connection"
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
            body="A verified address is needed before an app can connect to your account."
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
                <Link href="/settings">See connected apps</Link>
              </Button>
            }
            body={`Go back to ${stage.connection.appName}. The connection is approved, and the app can now finish connecting.`}
            mark={
              <Mark tone="accent">
                <Check aria-hidden="true" className="size-6" strokeWidth={2} />
              </Mark>
            }
            title={`${stage.connection.name} is connected`}
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
            body={`${stage.connection.appName} was not connected. The app will see that this request was denied.`}
            mark={
              <Mark tone="stop">
                <CircleX
                  aria-hidden="true"
                  className="size-6"
                  strokeWidth={1.6}
                />
              </Mark>
            }
            title="Connection declined"
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
    return (
      <Frame lede="Approve a private connection between your Illarin account and the app that showed you this code.">
        <Panel capture={capturePanel}>
          <ConnectionDecision
            connection={review.connection}
            deciding={stage.kind === "confirm" ? null : stage.decision}
            onCancel={startOver}
            onDecide={(decision) => {
              void decide(review, decision, setStage, setTrouble);
            }}
            trouble={trouble}
            userCode={review.source.userCode}
          />
        </Panel>
      </Frame>
    );
  }

  if (stage.kind === "loading") {
    return (
      <Frame lede="">
        <Panel busy capture={capturePanel}>
          <p className="font-ui text-ui text-mute">
            Loading the connection request…
          </p>
        </Panel>
      </Frame>
    );
  }

  return (
    <Frame lede="Start connecting in your app, then enter the code it shows.">
      <form
        noValidate
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          const userCode = typed.trim();
          if (!userCode) {
            setTrouble("Enter the code your app is showing.");
            return;
          }
          void loadReview(
            requestCode
              ? { kind: "authorization", requestCode, userCode }
              : { kind: "device", userCode },
          );
        }}
        ref={capturePanel}
        className="outline-none focus-visible:outline-none"
        tabIndex={-1}
      >
        <label
          className="font-display text-section font-medium tracking-tight text-ink"
          htmlFor="connect-code"
        >
          Type the code your app is showing
        </label>
        <input
          aria-describedby={
            trouble ? "connect-entry-trouble" : "connect-entry-reason"
          }
          autoCapitalize="characters"
          autoComplete="off"
          className="mt-5 block min-h-[4.5rem] w-full max-w-[26rem] rounded-plate border-0 bg-deep px-6 text-center font-mono text-[clamp(1.6rem,4.5vw,2.5rem)] tracking-[0.22em] text-ink uppercase outline-offset-2 placeholder:text-mute/45"
          enterKeyHint="go"
          id="connect-code"
          maxLength={12}
          name="code"
          onChange={(event) => setTyped(event.target.value.toUpperCase())}
          placeholder="XXXX-XXXX"
          required
          spellCheck={false}
          value={typed}
        />
        {trouble ? (
          <div className="mt-4" id="connect-entry-trouble">
            <Trouble>{trouble}</Trouble>
          </div>
        ) : null}
        <div className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-4">
          <Button size="large" type="submit" variant="primary">
            Review request
          </Button>
          <p
            className="min-w-0 max-w-[34ch] flex-1 font-prose text-meta text-mute"
            id="connect-entry-reason"
          >
            Next, review the app's name and the permissions it asks for.
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
          Connect an app
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
    ? `/v1/connect/authorizations/${encodeURIComponent(source.requestCode)}/${action}`
    : `/v1/connect/requests/${encodeURIComponent(source.userCode)}/${action}`;
  try {
    const { data, error, response } = await api<unknown>("POST", endpoint, {
      body: { approvalToken: source.approvalToken },
    });
    const answer = response.ok ? data : error;
    if (!response.ok) {
      setTrouble(
        refusalMessage(
          answer,
          decision === "approve"
            ? "This connection request could not be approved."
            : "This connection request could not be declined.",
        ),
      );
      setStage({ kind: "confirm", review });
      return;
    }

    if (isAuthorization) {
      if (
        !isConnectionRedirect(answer) ||
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
      setStage({ connection: review.connection, kind: "denied" });
      return;
    }

    if (!isPendingConnection(answer)) {
      setTrouble("Illarin returned an incomplete approval. Try again.");
      setStage({ kind: "confirm", review });
      return;
    }
    setStage({ connection: answer, kind: "approved" });
  } catch {
    setTrouble(UNREACHABLE);
    setStage({ kind: "confirm", review });
  }
}
