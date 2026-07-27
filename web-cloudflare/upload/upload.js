// presign-worker.js
// Cloudflare Worker: mint a batch of presigned R2 PUT URLs so a browser can
// upload files directly to R2. The Worker never touches the audio bytes — it
// only signs.
//
// AUTH CHANGE: reviewers no longer type a shared UPLOAD_TOKEN. Instead this
// endpoint sits behind **Cloudflare Access** (Zero Trust). Reviewers log in
// once via SSO or an email one-time PIN; Cloudflare injects a signed identity
// JWT on every request, and this Worker verifies it. Nothing to type, and you
// add/remove reviewers from the Access dashboard instead of sharing a secret.
//
// Deps:  npm install aws4fetch
//
// Required bindings (wrangler.toml [vars] + `wrangler secret put`):
//   R2_ACCOUNT_ID        - Cloudflare account id (the one in the R2 S3 endpoint)
//   R2_BUCKET            - target bucket name
//   R2_ACCESS_KEY_ID     - R2 S3 API access key id      (secret)
//   R2_SECRET_ACCESS_KEY - R2 S3 API secret access key  (secret)
//   ALLOWED_ORIGIN       - your app origin, e.g. https://qa.arti.example
//   ACCESS_TEAM_DOMAIN   - your Access team domain, e.g. myteam.cloudflareaccess.com
//   ACCESS_AUD           - the Application Audience (AUD) tag of the Access app
//   ALLOWED_EMAILS       - (optional) comma-separated extra allow-list of emails.
//                          Usually redundant with the Access policy, but a cheap
//                          second gate if you want it.
//
// SETUP (one time):
//   1. Zero Trust dashboard -> Access -> Applications -> Add a self-hosted app.
//      Domain: qa.arti.example  Path: /api/presign-batch  (or protect /api/*).
//   2. Add a policy: Allow, Include -> Emails / Email domain / IdP group for your
//      reviewers. Pick a login method (Google, Microsoft, GitHub, or One-time PIN).
//   3. Copy the Application Audience (AUD) tag into ACCESS_AUD.
//   4. Set ACCESS_TEAM_DOMAIN to <yourteam>.cloudflareaccess.com.
//   Because the route is behind Access, Cloudflare adds the verified JWT as the
//   `Cf-Access-Jwt-Assertion` header automatically before it reaches this Worker.
//
// R2 CORS still required: the R2 *bucket* needs a CORS policy allowing PUT from
// ALLOWED_ORIGIN, or the browser blocks the direct upload even with a valid URL.

import { AwsClient } from "aws4fetch";

const PRESIGN_EXPIRES = 3600; // seconds each URL stays valid
const MAX_BATCH = 1000;       // guard against absurd requests
const MINT_PATH = "/api/presign-batch";

export default {
  async fetch(request, env) {
    const origin = request.headers.get("Origin") || "";
    const corsOrigin = origin && origin === env.ALLOWED_ORIGIN ? origin : (env.ALLOWED_ORIGIN || "");

    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers: corsHeaders(corsOrigin) });
    }

    const { pathname } = new URL(request.url);
    if (pathname !== MINT_PATH || request.method !== "POST") {
      return new Response("Not found", { status: 404 });
    }

    // --- Authorization: verify the Cloudflare Access identity JWT. ---
    const identity = await verifyAccess(request, env);
    if (!identity.ok) {
      return json({ error: "unauthorized", reason: identity.reason }, 401, corsOrigin);
    }

    let payload;
    try {
      payload = await request.json();
    } catch {
      return json({ error: "invalid JSON body" }, 400, corsOrigin);
    }

    const prefix = typeof payload.prefix === "string" ? payload.prefix : "";
    const paths = payload.paths;
    if (!Array.isArray(paths) || paths.length === 0) {
      return json({ error: "paths[] is required" }, 400, corsOrigin);
    }
    if (paths.length > MAX_BATCH) {
      return json({ error: `too many paths (max ${MAX_BATCH})` }, 400, corsOrigin);
    }

    const aws = new AwsClient({
      accessKeyId: env.R2_ACCESS_KEY_ID,
      secretAccessKey: env.R2_SECRET_ACCESS_KEY,
      service: "s3",
      region: "auto", // R2 uses the "auto" region
    });

    const endpoint = `https://${env.R2_ACCOUNT_ID}.r2.cloudflarestorage.com`;

    const results = await Promise.all(
      paths.map(async (rel) => {
        const key = buildKey(prefix, rel);
        const target =
          `${endpoint}/${env.R2_BUCKET}/${encodeKey(key)}?X-Amz-Expires=${PRESIGN_EXPIRES}`;
        const signed = await aws.sign(target, { method: "PUT", aws: { signQuery: true } });
        return { path: rel, key, url: signed.url };
      })
    );

    // Optionally record who minted, for an audit trail.
    return json({ mintedBy: identity.email, expiresIn: PRESIGN_EXPIRES, results }, 200, corsOrigin);
  },
};

// --- Cloudflare Access verification ----------------------------------------

// Verify the Access identity JWT. Returns { ok, email } or { ok:false, reason }.
// The token arrives either as the `Cf-Access-Jwt-Assertion` header (injected by
// Cloudflare when the route is behind Access) or the `CF_Authorization` cookie.
async function verifyAccess(request, env) {
  const token =
    request.headers.get("Cf-Access-Jwt-Assertion") || cookie(request, "CF_Authorization");
  if (!token) return { ok: false, reason: "no access token (is the route behind Access?)" };

  const parts = token.split(".");
  if (parts.length !== 3) return { ok: false, reason: "malformed token" };
  const [headerB64, payloadB64, sigB64] = parts;

  let header, claims;
  try {
    header = JSON.parse(b64urlToText(headerB64));
    claims = JSON.parse(b64urlToText(payloadB64));
  } catch {
    return { ok: false, reason: "unparseable token" };
  }
  if (header.alg !== "RS256") return { ok: false, reason: `unexpected alg ${header.alg}` };

  // Claim checks (do the cheap ones before the crypto).
  const now = Math.floor(Date.now() / 1000);
  if (typeof claims.exp === "number" && now >= claims.exp) return { ok: false, reason: "expired" };
  if (typeof claims.nbf === "number" && now < claims.nbf) return { ok: false, reason: "not yet valid" };

  const expectedIss = `https://${env.ACCESS_TEAM_DOMAIN}`;
  if (claims.iss !== expectedIss) return { ok: false, reason: "issuer mismatch" };

  const aud = Array.isArray(claims.aud) ? claims.aud : [claims.aud];
  if (!aud.includes(env.ACCESS_AUD)) return { ok: false, reason: "audience mismatch" };

  // Signature check against the team's public keys.
  const jwk = await getSigningKey(env.ACCESS_TEAM_DOMAIN, header.kid);
  if (!jwk) return { ok: false, reason: "unknown signing key" };

  const key = await crypto.subtle.importKey(
    "jwk", jwk, { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" }, false, ["verify"]
  );
  const signed = new TextEncoder().encode(`${headerB64}.${payloadB64}`);
  const valid = await crypto.subtle.verify(
    "RSASSA-PKCS1-v1_5", key, b64urlToBytes(sigB64), signed
  );
  if (!valid) return { ok: false, reason: "bad signature" };

  // Optional extra allow-list on top of the Access policy.
  const email = (claims.email || "").toLowerCase();
  if (env.ALLOWED_EMAILS) {
    const allowed = env.ALLOWED_EMAILS.split(",").map((e) => e.trim().toLowerCase()).filter(Boolean);
    if (!allowed.includes(email)) return { ok: false, reason: "email not allow-listed" };
  }

  return { ok: true, email };
}

// Fetch and cache the Access JWKS (public keys). Cached in-memory per isolate
// with a short TTL; Cloudflare rotates these keys periodically.
const _jwksCache = new Map(); // teamDomain -> { keys, fetchedAt }
const JWKS_TTL_MS = 60 * 60 * 1000; // 1 hour

async function getSigningKey(teamDomain, kid) {
  const cached = _jwksCache.get(teamDomain);
  const fresh = cached && Date.now() - cached.fetchedAt < JWKS_TTL_MS;
  let keys = fresh ? cached.keys : null;

  if (!keys) {
    const res = await fetch(`https://${teamDomain}/cdn-cgi/access/certs`);
    if (!res.ok) return null;
    const body = await res.json();
    keys = body.keys || [];
    _jwksCache.set(teamDomain, { keys, fetchedAt: Date.now() });
  }

  let jwk = keys.find((k) => k.kid === kid);
  // If the kid wasn't found, the keys may have just rotated — refetch once.
  if (!jwk && fresh) {
    const res = await fetch(`https://${teamDomain}/cdn-cgi/access/certs`);
    if (res.ok) {
      const body = await res.json();
      keys = body.keys || [];
      _jwksCache.set(teamDomain, { keys, fetchedAt: Date.now() });
      jwk = keys.find((k) => k.kid === kid);
    }
  }
  return jwk || null;
}

// --- helpers ---------------------------------------------------------------

function cookie(request, name) {
  const raw = request.headers.get("Cookie") || "";
  for (const part of raw.split(";")) {
    const [k, ...v] = part.trim().split("=");
    if (k === name) return v.join("=");
  }
  return "";
}

function b64urlToBytes(s) {
  const pad = s.length % 4 ? "=".repeat(4 - (s.length % 4)) : "";
  const b64 = s.replace(/-/g, "+").replace(/_/g, "/") + pad;
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function b64urlToText(s) {
  return new TextDecoder().decode(b64urlToBytes(s));
}

// Join a prefix and a relative path into a clean R2 key.
function buildKey(prefix, rel) {
  const clean = String(rel).replace(/^\.?\/+/, "").replace(/\/{2,}/g, "/");
  const p = prefix.replace(/^\/+|\/+$/g, "");
  return p ? `${p}/${clean}` : clean;
}

// Percent-encode each path segment but keep the slashes as key separators.
function encodeKey(key) {
  return key.split("/").map(encodeURIComponent).join("/");
}

function corsHeaders(origin) {
  return {
    "Access-Control-Allow-Origin": origin,
    "Access-Control-Allow-Methods": "POST, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
    "Access-Control-Allow-Credentials": "true", // so the Access cookie rides along
    "Access-Control-Max-Age": "86400",
    "Vary": "Origin",
  };
}

function json(obj, status, origin) {
  return new Response(JSON.stringify(obj), {
    status,
    headers: { "Content-Type": "application/json", ...corsHeaders(origin) },
  });
}
