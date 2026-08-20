// viewer.js
// Cloudflare Worker serving the viewer page: a read-only browser over the
// arti-input, arti-models, and arti-output R2 buckets. Uses native R2 bucket
// bindings (see viewer.toml) rather than signed S3 API calls - listing is the
// bulk of this Worker's job, and bindings return plain objects instead of the
// XML ListObjectsV2 responses the S3-compatible endpoint would require
// hand-parsing.
//
// Routes:
//   GET  /                    - the page itself
//   GET  /api/models          - [{modelType, langIso, highestRunNum, runs: {runNum: uploaded}, uploaded}]
//   GET  /api/models/details  - [{filename, uploaded, key}] for one model run's processor_<langIso>/ files
//   GET  /api/models/download - zip of one model run: <langIso>/<tensor file>, <langIso>/processor_<langIso>/...
//   GET  /api/input           - [{mediaId, uploaded}]
//   GET  /api/input/:id       - [{prefix, filename, uploaded, action, key}]
//   GET  /api/output          - [{username, mediaId, module, highestRunNum}]
//   GET  /api/output/details  - fixed-row (label, value, viewMode, viewKey, downloadKey)
//   POST /api/output/notes    - {username, mediaId, module, runNum, text} - writes the run's notes object
//   GET  /file                - streams/downloads/opens a single R2 object; accepts
//                               an optional exp+sig pair (see signFileParams) that
//                               authorizes the request on its own, regardless of mode
//   GET  /api/sign-audio-urls - {bucket, key} of a report's audio_file_urls.json
//                               manifest (see generic/audio_file.go) -> that manifest
//                               with each entry's signed_url filled in with a 7-day
//                               signed /file link, for swapping into a downloaded
//                               report's audio buttons client-side
//
// No auth is applied here - the original spec didn't call for it. These
// buckets contain internal pipeline input/output, so consider putting this
// route behind Cloudflare Access the same way web-cloudflare/upload does,
// before exposing it beyond localhost/VPN. If Access is added later, /file
// requests carrying a valid sig must stay exempt from it - that signature is
// what lets a downloaded, forwarded report keep playing its audio for
// someone with no Access identity at all; see AUDIO_SIGN_SECRET below.

import { PAGE_HTML } from "./viewer_page.js";
import {
  getModelsRows,
  getModelDetailRows,
  getInputRows,
  getInputDetailRows,
  getOutputRows,
} from "./r2_list.js";
import { getOutputDetailRows, outputRunPrefix } from "./output_details.js";
import { buildModelZip } from "./model_zip.js";

const BUCKET_NAMES = {
  input: "arti-input",
  models: "arti-models",
  output: "arti-output",
};

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const path = url.pathname;

    try {
      if (path === "/") {
        return new Response(PAGE_HTML, { headers: { "content-type": "text/html; charset=utf-8" } });
      }
      if (path === "/api/models") {
        return json(await getModelsRows(env.ARTI_MODELS));
      }
      if (path === "/api/input") {
        return json(await getInputRows(env.ARTI_INPUT));
      }
      if (path.startsWith("/api/input/")) {
        const mediaId = decodeURIComponent(path.slice("/api/input/".length));
        if (!mediaId) return json({ error: "media_id required" }, 400);
        return json(await getInputDetailRows(env.ARTI_INPUT, mediaId));
      }
      if (path === "/api/output") {
        return json(await getOutputRows(env.ARTI_OUTPUT));
      }
      if (path === "/api/output/details") {
        const { username, mediaId, module, runNum } = Object.fromEntries(url.searchParams);
        if (!username || !mediaId || !module || !runNum) {
          return json({ error: "username, mediaId, module, runNum are all required" }, 400);
        }
        return json(await getOutputDetailRows(env.ARTI_OUTPUT, username, mediaId, module, runNum));
      }
      if (path === "/api/output/notes") {
        if (request.method !== "POST") return new Response("Method not allowed", { status: 405 });
        const { username, mediaId, module, runNum, text } = await request.json();
        if (!username || !mediaId || !module || !runNum) {
          return json({ error: "username, mediaId, module, runNum are all required" }, 400);
        }
        const key = `${outputRunPrefix(username, mediaId, module, runNum)}notes`;
        await env.ARTI_OUTPUT.put(key, text || "");
        return json({ ok: true });
      }
      if (path === "/file") {
        return await serveFile(request, url, env);
      }
      if (path === "/api/sign-audio-urls") {
        const bucketParam = url.searchParams.get("bucket");
        const key = url.searchParams.get("key");
        if (!bucketParam || !key) return json({ error: "bucket and key are required" }, 400);
        return await signAudioUrls(env, url, bucketParam, key);
      }
      if (path === "/api/models/details") {
        const { modelType, langIso, runNum } = Object.fromEntries(url.searchParams);
        if (!modelType || !langIso || !runNum) {
          return json({ error: "modelType, langIso, runNum are all required" }, 400);
        }
        return json(await getModelDetailRows(env.ARTI_MODELS, modelType, langIso, runNum));
      }
      if (path === "/api/models/download") {
        const { modelType, langIso, runNum } = Object.fromEntries(url.searchParams);
        if (!modelType || !langIso || !runNum) {
          return json({ error: "modelType, langIso, runNum are all required" }, 400);
        }
        return await serveModelZip(env.ARTI_MODELS, modelType, langIso, runNum);
      }
      return new Response("Not found", { status: 404 });
    } catch (err) {
      return json({ error: String((err && err.message) || err) }, 500);
    }
  },
};

// Text preview cap for mode=show. The front end's "View" dialog reads the
// whole response into one JS string (res.text()) and drops it into a <pre>,
// which is fine for a config/log file but would hang or crash the tab on a
// truly huge one - so past this many bytes the Worker itself truncates and
// appends a notice, rather than shipping the full file into the browser.
// mode=open (real .html/.htm pages, opened in their own tab) is NOT capped:
// the browser renders those as a normal page load, never buffering the body
// into a JS string, so the same risk doesn't apply.
const SHOW_PREVIEW_LIMIT = 2 * 1024 * 1024; // 2 MB

// Streams a single object out of one of the three buckets. `mode` controls
// the response: download forces a Save As with a flattened filename, show
// returns a (possibly truncated) plain-text preview, open serves the object
// inline with an HTML content-type so a new tab renders it as a real page,
// play streams audio with Range support so <audio> seeking works.
async function serveFile(request, url, env) {
  const bucketParam = url.searchParams.get("bucket");
  const key = url.searchParams.get("key") || "";
  const mode = url.searchParams.get("mode") || "download";

  const bucket = { input: env.ARTI_INPUT, models: env.ARTI_MODELS, output: env.ARTI_OUTPUT }[bucketParam];
  if (!bucket) return new Response("Unknown bucket", { status: 400 });
  if (!key || key.startsWith("/") || key.includes("..")) {
    return new Response("Invalid key", { status: 400 });
  }

  // Plain relative /file links (no sig) are how the page itself always
  // fetches - those are unaffected. A sig is only ever present on a signed
  // link minted by signAudioUrls for a downloaded report, and once present
  // it must check out: a present-but-bad/expired sig should not silently
  // fall back to unauthenticated access.
  const sig = url.searchParams.get("sig");
  if (sig) {
    const ok = await verifyFileParams(env, bucketParam, key, url.searchParams.get("exp"), sig);
    if (!ok) return new Response("Invalid or expired signature", { status: 403 });
  }

  if (mode === "play") return streamWithRange(request, bucket, key);
  if (mode === "show") return serveTextPreview(bucket, key, url.searchParams.get("tail") === "1");

  const object = await bucket.get(key);
  if (!object) return new Response("Not found", { status: 404 });

  const headers = new Headers();
  object.writeHttpMetadata(headers);
  headers.set("etag", object.httpEtag);

  if (mode === "open") {
    // mode=open is only ever requested for files classifyView() already
    // identified as .html/.htm, so force the type rather than trusting
    // R2's stored content-type - uploads that didn't set one explicitly
    // come back as application/octet-stream, which browsers won't render.
    headers.set("content-type", "text/html; charset=utf-8");
    headers.set("content-disposition", "inline");
    return new Response(object.body, { headers });
  }

  // download: force Save As with the full path flattened into the filename,
  // e.g. arti-input/N2XNRPMS/foo.wav -> arti-input_N2XNRPMS_foo.wav
  const flatName = `${BUCKET_NAMES[bucketParam]}_${key.split("/").join("_")}`;
  headers.set("content-disposition", `attachment; filename="${flatName}"`);
  return new Response(object.body, { headers });
}

// Reads a report's audio_file_urls.json manifest (written by
// generic.OutputAudioFiles, uploaded next to the report under the same
// .../output/ run prefix) and returns it with every entry's signed_url
// filled in - a 7-day signed /file link, matching the expiry the old
// generation-time SignAudioURL used. The signature is a bearer credential
// (see viewer.js's route comment): anyone holding the URL can use it until
// exp, with no session or Access identity required, which is the whole
// point - it's what lets a downloaded report keep playing audio wherever
// it's forwarded.
const SIGNED_URL_TTL_SECONDS = 7 * 24 * 60 * 60;

async function signAudioUrls(env, url, bucketParam, manifestKey) {
  if (!env.AUDIO_SIGN_SECRET) return json({ error: "AUDIO_SIGN_SECRET is not configured" }, 500);
  const bucket = { input: env.ARTI_INPUT, models: env.ARTI_MODELS, output: env.ARTI_OUTPUT }[bucketParam];
  if (!bucket) return json({ error: "Unknown bucket" }, 400);
  const object = await bucket.get(manifestKey);
  if (!object) return json({ error: "Manifest not found" }, 404);
  const entries = await object.json();

  const exp = Math.floor(Date.now() / 1000) + SIGNED_URL_TTL_SECONDS;
  const results = [];
  for (const entry of entries) {
    if (!entry.unsigned_url) continue;
    // unsigned_url (from generic.AudioFile) is a relative /file?bucket=...&key=...
    // path - resolve it against this request's own origin so bucket/key can be
    // read back out, and so the signed link we build is absolute (a downloaded
    // report has no origin of its own to resolve a relative link against).
    const fileUrl = new URL(entry.unsigned_url, url.origin);
    const fileBucket = fileUrl.searchParams.get("bucket");
    const fileKey = fileUrl.searchParams.get("key");
    if (!fileBucket || !fileKey) continue;
    const sig = await signFileParams(env, fileBucket, fileKey, exp);
    fileUrl.searchParams.set("exp", String(exp));
    fileUrl.searchParams.set("sig", sig);
    results.push({ unsigned_url: entry.unsigned_url, signed_url: fileUrl.toString() });
  }
  return json(results);
}

async function signFileParams(env, bucket, key, exp) {
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(env.AUDIO_SIGN_SECRET),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sigBytes = await crypto.subtle.sign("HMAC", cryptoKey, new TextEncoder().encode(`${bucket}\n${key}\n${exp}`));
  return [...new Uint8Array(sigBytes)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function verifyFileParams(env, bucket, key, exp, sig) {
  if (!env.AUDIO_SIGN_SECRET || !exp || !sig) return false;
  if (Date.now() / 1000 > Number(exp)) return false;
  const expected = await signFileParams(env, bucket, key, exp);
  return timingSafeEqual(expected, sig);
}

function timingSafeEqual(a, b) {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}

// Bundles one model (modelType/langIso) into a zip whose single top-level
// folder is the langIso code, containing the tensor file and the
// processor_<langIso>/ tokenizer directory as they're laid out in R2 - so
// unzipping drops in exactly the folder a local model dir expects.
async function serveModelZip(bucket, modelType, langIso, runNum) {
  const zipBytes = await buildModelZip(bucket, modelType, langIso, runNum);
  if (!zipBytes) return new Response("Not found", { status: 404 });
  return new Response(zipBytes, {
    headers: {
      "content-type": "application/zip",
      "content-disposition": `attachment; filename="${langIso}.zip"`,
    },
  });
}

// Serves mode=show: the full object streamed as-is when it's under
// SHOW_PREVIEW_LIMIT, otherwise a bounded SHOW_PREVIEW_LIMIT-byte R2 range
// read (not the whole file) with a truncation notice appended - the first
// bytes by default, or the last bytes when tail=1 (log files bury the
// interesting part - the error - at the end, past where a head truncation
// would ever reach). Only that bounded slice is ever buffered into a string
// here.
async function serveTextPreview(bucket, key, tail) {
  const head = await bucket.head(key);
  if (!head) return new Response("Not found", { status: 404 });

  const headers = new Headers();
  head.writeHttpMetadata(headers);
  headers.set("content-type", "text/plain; charset=utf-8");

  if (head.size <= SHOW_PREVIEW_LIMIT) {
    const object = await bucket.get(key);
    return new Response(object.body, { headers });
  }

  if (tail) {
    const object = await bucket.get(key, { range: { suffix: SHOW_PREVIEW_LIMIT } });
    const text = await object.text();
    const notice =
      `[... showing the last ${formatBytes(SHOW_PREVIEW_LIMIT)} of ` +
      `${formatBytes(head.size)}. Use Download for the complete file. ...]\n\n`;
    return new Response(notice + text, { headers });
  }

  const object = await bucket.get(key, { range: { offset: 0, length: SHOW_PREVIEW_LIMIT } });
  const text = await object.text();
  const notice =
    `\n\n[... truncated - showing the first ${formatBytes(SHOW_PREVIEW_LIMIT)} of ` +
    `${formatBytes(head.size)}. Use Download for the complete file. ...]`;
  return new Response(text + notice, { headers });
}

function formatBytes(bytes) {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// Range support for the <audio> "play" action. The browser drives seeking via
// standard Range requests; without honoring them, <audio> can only play from
// the start and can't seek.
async function streamWithRange(request, bucket, key) {
  const head = await bucket.head(key);
  if (!head) return new Response("Not found", { status: 404 });
  const totalSize = head.size;

  const rangeHeader = request.headers.get("range");
  const headers = new Headers();
  head.writeHttpMetadata(headers);

  if (!rangeHeader) {
    const object = await bucket.get(key);
    headers.set("accept-ranges", "bytes");
    headers.set("content-length", String(totalSize));
    return new Response(object.body, { headers });
  }

  const match = /^bytes=(\d+)-(\d+)?$/.exec(rangeHeader);
  if (!match) return new Response("Invalid Range", { status: 416 });
  const start = parseInt(match[1], 10);
  const end = match[2] ? parseInt(match[2], 10) : totalSize - 1;
  if (start >= totalSize || end >= totalSize || start > end) {
    headers.set("content-range", `bytes */${totalSize}`);
    return new Response(null, { status: 416, headers });
  }

  const object = await bucket.get(key, { range: { offset: start, length: end - start + 1 } });
  headers.set("accept-ranges", "bytes");
  headers.set("content-range", `bytes ${start}-${end}/${totalSize}`);
  headers.set("content-length", String(end - start + 1));
  return new Response(object.body, { status: 206, headers });
}

function json(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" },
  });
}
