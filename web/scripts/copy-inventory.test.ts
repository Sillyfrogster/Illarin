import { expect, test } from "bun:test";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { inventory } from "./copy-inventory";

test("inventories source copy and templates once with all locations", async () => {
  const root = await mkdtemp(join(tmpdir(), "illarin-copy-"));
  try {
    await mkdir(join(root, "web/src/app"), { recursive: true });
    await mkdir(join(root, "api/internal/account"), { recursive: true });
    await mkdir(join(root, "web/public"), { recursive: true });
    await writeFile(
      join(root, "web/public/mark.svg"),
      "<svg><title>Site mark</title><text>Welcome</text></svg>",
    );
    await writeFile(
      join(root, "web/src/app/page.tsx"),
      `import { Button } from "ignored-import";
const metadata = { title: "Account", description: "Read your account details." };
const labels = { ready: "Ready", waiting: "waiting", name: "Visits" };
const message = \`Hello \${name}, you have \${count} notices.\`;
const technical = status === "ignored-comparison";
buffer.toString("ignored-encoding");
api("ignored-method", "/example");
type State = "ignored-type";
export default () => <main className="ignored-class">
  <h1>Read &amp; write</h1><Button title="Save your work">Save</Button>
  <input placeholder="Your name" aria-label="Name" />
  <p className={busy ? "ignored-conditional-class" : ""}>{busy ? "Saving…" : "Save"}</p><button>×</button>
</main>;`,
    );
    await writeFile(
      join(root, "api/internal/account/messages.go"),
      `package account
import "ignored-go-import"
// "ignored-comment"
type Record struct { Name string \`json:"ignored-tag"\` }
func send(link string) {
  refuse("Try again.")
  refuse("Ask for work:receive, library:sync, or both, once each.")
  mail("Verify your email", "Open this link: " + link)
  c.Header("ignored-header", "ignored-header-value")
}
var label = "Save"
var query = "SELECT name FROM users"
`,
    );
    await writeFile(
      join(root, "web/src/app/page.test.tsx"),
      'const fixture = "ignored-test";',
    );
    const result = await inventory(root);
    for (const copy of [
      "Account",
      "Read your account details.",
      "Ready",
      "waiting",
      `Hello \${name}, you have \${count} notices.`,
      "Visits",
      "Site mark",
      "Welcome",
      "`×`",
      "Read & write",
      "Save your work",
      "Your name",
      "Name",
      "Saving…",
      "Try again.",
      "Ask for work:receive, library:sync, or both, once each.",
      "Verify your email",
      `Open this link: \${link}`,
    ]) {
      expect(result).toContain(copy);
    }
    expect(result.match(/^- \[ \] `Save`/gm)).toHaveLength(1);
    expect(result).toContain("web/src/app/page.tsx:10");
    expect(result).toContain("api/internal/account/messages.go:11");
    expect(result).not.toContain("ignored-");
    expect(result).not.toContain("SELECT name");
    expect(result).toContain("Sweep ticket:");
    expect(await inventory(root)).toBe(result);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}, 60_000);
