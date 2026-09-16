// Sets the Umami admin password and creates the site's website

const origin = "http://127.0.0.1:3000";
const defaultPassword = "umami";
const { UMAMI_ADMIN_PASSWORD: password, ANALYTICS_WEBSITE_ID: websiteId, ANALYTICS_DOMAIN: domain } = process.env;

async function call(method, path, token, body) {
  const response = await fetch(`${origin}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await response.text();
  return { status: response.status, body: text ? JSON.parse(text) : null };
}

async function signIn(attempt) {
  const answer = await call("POST", "/api/auth/login", null, { username: "admin", password: attempt });
  return answer.status === 200 ? answer.body.token : null;
}

async function adminToken() {
  const token = await signIn(password);
  if (token) return token;
  const first = await signIn(defaultPassword);
  if (!first) {
    throw new Error("Umami refused both UMAMI_ADMIN_PASSWORD and its first-run password.");
  }
  const changed = await call("POST", "/api/me/password", first, {
    currentPassword: defaultPassword,
    newPassword: password,
  });
  if (changed.status !== 200) {
    throw new Error(`Umami did not accept the new admin password: ${changed.status}`);
  }
  console.log("Set the Umami admin password.");
  return signIn(password);
}

async function ensureWebsite(token) {
  const found = await call("GET", `/api/websites/${websiteId}`, token);
  if (found.status === 200 && found.body) return;
  const created = await call("POST", "/api/websites", token, { id: websiteId, name: "Illarin", domain });
  if (created.status !== 200) {
    throw new Error(`Umami did not create the website: ${created.status}`);
  }
  console.log(`Created the Umami website for ${domain}.`);
}

if (!password || password.length < 8) {
  throw new Error("UMAMI_ADMIN_PASSWORD must be at least 8 characters.");
}
if (!websiteId || !domain) {
  throw new Error("ANALYTICS_WEBSITE_ID and ANALYTICS_DOMAIN are required.");
}
await ensureWebsite(await adminToken());
