// viewer.js
// Cloudflare Worker serving the /viewer page: a read-only browser over the
// arti-input, arti-models, and arti-output R2 buckets. Uses native R2 bucket
// bindings (see viewer.toml) rather than signed S3 API calls - listing is the
// bulk of this Worker's job, and bindings return plain objects instead of the
// XML ListObjectsV2 responses the S3-compatible endpoint would require
// hand-parsing.
//
// Routes:
//   GET  /viewer                    - the page itself
//   GET  /viewer/api/models         - [{modelType, langIso, uploaded}]
//   GET  /viewer/api/input          - [mediaId, ...]
//   GET  /viewer/api/input/:id      - [{prefix, filename, uploaded, action, key}]
//   GET  /viewer/api/output         - [{username, mediaId, module, highestRunNum}]
//   GET  /viewer/api/output/details - fixed-row (label, value, viewMode, viewKey, downloadKey)
//   GET  /viewer/file               - streams/downloads/opens a single R2 object
//
// No auth is applied here - the original spec didn't call for it. These
// buckets contain internal pipeline input/output, so consider putting this
// route behind Cloudflare Access the same way web-cloudflare/upload does,
// before exposing it beyond localhost/VPN.

import { PAGE_HTML } from "./viewer_page.js";
import {
  getModelsRows,
  getInputMediaIds,
  getInputDetailRows,
  getOutputRows,
} from "./r2_list.js";
import { getOutputDetailRows } from "./output_details.js";

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
      if (path === "/viewer" || path === "/viewer/") {
        return new Response(PAGE_HTML, { headers: { "content-type": "text/html; charset=utf-8" } });
      }
      if (path === "/viewer/api/models") {
        return json(await getModelsRows(env.ARTI_MODELS));
      }
      if (path === "/viewer/api/input") {
        return json(await getInputMediaIds(env.ARTI_INPUT));
      }
      if (path.startsWith("/viewer/api/input/")) {
        const mediaId = decodeURIComponent(path.slice("/viewer/api/input/".length));
        if (!mediaId) return json({ error: "media_id required" }, 400);
        return json(await getInputDetailRows(env.ARTI_INPUT, mediaId));
      }
      if (path === "/viewer/api/output") {
        return json(await getOutputRows(env.ARTI_OUTPUT));
      }
      if (path === "/viewer/api/output/details") {
        const { username, mediaId, module, runNum } = Object.fromEntries(url.searchParams);
        if (!username || !mediaId || !module || !runNum) {
          return json({ error: "username, mediaId, module, runNum are all required" }, 400);
        }
        return json(await getOutputDetailRows(env.ARTI_OUTPUT, username, mediaId, module, runNum));
      }
      if (path === "/viewer/file") {
        return await serveFile(request, url, env);
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

  if (mode === "play") return streamWithRange(request, bucket, key);
  if (mode === "show") return serveTextPreview(bucket, key);

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

// Serves mode=show: the full object streamed as-is when it's under
// SHOW_PREVIEW_LIMIT, otherwise just the first SHOW_PREVIEW_LIMIT bytes (a
// bounded R2 range read, not the whole file) with a truncation notice
// appended. Only that bounded prefix is ever buffered into a string here.
async function serveTextPreview(bucket, key) {
  const head = await bucket.head(key);
  if (!head) return new Response("Not found", { status: 404 });

  const headers = new Headers();
  head.writeHttpMetadata(headers);
  headers.set("content-type", "text/plain; charset=utf-8");

  if (head.size <= SHOW_PREVIEW_LIMIT) {
    const object = await bucket.get(key);
    return new Response(object.body, { headers });
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
