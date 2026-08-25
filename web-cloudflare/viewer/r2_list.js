// r2_list.js
// Helpers for paginating R2 bucket.list() and aggregating keys into the
// distinct-tuple rows the /viewer tables need. Kept separate from viewer.js
// so the routing/streaming code isn't tangled up with key-parsing logic.

const LIST_PAGE_LIMIT = 1000;

// Walks bucket.list({ prefix }) to completion (no delimiter), returning every
// R2Object under prefix. Used instead of one list() call per distinct group
// so aggregation stays O(objectCount / 1000) R2 ops rather than O(groupCount).
export async function listAllObjects(bucket, prefix) {
  const objects = [];
  let cursor;
  for (;;) {
    const page = await bucket.list({ prefix, cursor, limit: LIST_PAGE_LIMIT });
    objects.push(...page.objects);
    if (!page.truncated) break;
    cursor = page.cursor;
  }
  return objects;
}

const AUDIO_EXTENSIONS = [".wav", ".mp3", ".m4a", ".flac", ".ogg", ".aac"];
const HTML_EXTENSIONS = [".html", ".htm"];
const TEXT_VIEW_EXTENSIONS = [".usx", ".usfm", ".sfm", ".xml"];

// Classifies the *primary* action button a file row should show - play,
// open (renders as an actual page in a new tab; for .html/.htm, which have
// their own markup/styling that a plain-text dialog would just show as raw
// source), or show (plain-text preview dialog for markup-ish-but-not-HTML
// formats) - independent of this, every row also gets a Download button
// (see viewer_page.js), so this returns null for anything else rather than
// a "download" action of its own.
export function classifyView(filename) {
  const lower = filename.toLowerCase();
  if (AUDIO_EXTENSIONS.some((ext) => lower.endsWith(ext))) return "play";
  if (HTML_EXTENSIONS.some((ext) => lower.endsWith(ext))) return "open";
  if (TEXT_VIEW_EXTENSIONS.some((ext) => lower.endsWith(ext))) return "show";
  return null;
}

// Run numbers are stored zero-padded to 5 digits in R2 keys
// (mms_adapters/{iso}/00001/...); the UI/API deal in plain integers, so this
// is the single place that converts (mirrors output_details.js's
// outputRunPrefix convention).
export function padRunNum(runNum) {
  return String(runNum).padStart(5, "0");
}

// arti-models: {model_type}/{lang_iso}/{run_num}/{various model files}
// One row per distinct (model_type, lang_iso), tracking every run_num seen
// (runs: {runNum: uploaded}) so the UI can switch between them; "uploaded"
// defaults to whichever run is highest.
export async function getModelsRows(bucket) {
  const objects = await listAllObjects(bucket, "");
  const groups = new Map();
  for (const obj of objects) {
    const parts = obj.key.split("/");
    if (parts.length < 3) continue; // malformed/stray key, skip
    const [modelType, langIso, runNumStr] = parts;
    const runNum = parseInt(runNumStr, 10);
    if (Number.isNaN(runNum)) continue;
    const groupKey = `${modelType} ${langIso}`;
    let group = groups.get(groupKey);
    if (!group) {
      group = { modelType, langIso, highestRunNum: runNum, runs: {} };
      groups.set(groupKey, group);
    }
    if (!(runNum in group.runs)) group.runs[runNum] = obj.uploaded;
    if (runNum > group.highestRunNum) group.highestRunNum = runNum;
  }
  const rows = Array.from(groups.values()).map((g) => ({ ...g, uploaded: g.runs[g.highestRunNum] }));
  rows.sort((a, b) => a.modelType.localeCompare(b.modelType) || a.langIso.localeCompare(b.langIso));
  return rows;
}

// arti-models processor files for one (model_type, lang_iso, run_num): the
// tokenizer files under processor_<lang_iso>/, one row per file.
export async function getModelDetailRows(bucket, modelType, langIso, runNum) {
  const prefix = `${modelType}/${langIso}/${padRunNum(runNum)}/processor_${langIso}/`;
  const objects = await listAllObjects(bucket, prefix);
  const rows = [];
  for (const obj of objects) {
    const filename = obj.key.slice(prefix.length);
    if (!filename) continue; // directory placeholder object, not a real file
    rows.push({ filename, uploaded: obj.uploaded, key: obj.key });
  }
  rows.sort((a, b) => a.filename.localeCompare(b.filename));
  return rows;
}

// arti-input top level: {media_id}/{various paths}/{filename}
// One row per distinct media_id; uploaded is taken from whichever file under
// that media_id is encountered first (same convention as getModelsRows).
export async function getInputRows(bucket) {
  const objects = await listAllObjects(bucket, "");
  const groups = new Map();
  for (const obj of objects) {
    const slash = obj.key.indexOf("/");
    if (slash === -1) continue; // malformed/stray key, skip
    const mediaId = obj.key.slice(0, slash);
    if (!groups.has(mediaId)) {
      groups.set(mediaId, { mediaId, uploaded: obj.uploaded });
    }
  }
  const rows = Array.from(groups.values());
  rows.sort((a, b) => a.mediaId.localeCompare(b.mediaId));
  return rows;
}

// arti-input details for one media_id: the 1-3 element path between
// media_id/ and the filename is displayed as a single '/'-joined prefix
// column (per user's decision), one row per distinct (prefix, filename).
export async function getInputDetailRows(bucket, mediaId) {
  const prefix = `${mediaId}/`;
  const objects = await listAllObjects(bucket, prefix);
  const seen = new Set();
  const rows = [];
  for (const obj of objects) {
    const rest = obj.key.slice(prefix.length);
    const segments = rest.split("/");
    const filename = segments.pop();
    if (!filename) continue; // directory placeholder object, not a real file
    const prefixDisplay = segments.join("/");
    const dedupeKey = `${prefixDisplay} ${filename}`;
    if (seen.has(dedupeKey)) continue;
    seen.add(dedupeKey);
    rows.push({
      prefix: prefixDisplay,
      filename,
      uploaded: obj.uploaded,
      key: obj.key,
      action: classifyView(filename),
    });
  }
  rows.sort((a, b) => a.prefix.localeCompare(b.prefix) || a.filename.localeCompare(b.filename));
  return rows;
}

// arti-output top level: {username}/{media_id}/{module}/{run_num}/{file_type}/{file_content}
// One row per distinct (username, media_id, module), tracking every run_num
// seen (runs: [runNum, ...], ascending) so the UI can offer only run numbers
// that actually still exist - runs get deleted individually, so the range
// from 1 to highestRunNum can have gaps.
export async function getOutputRows(bucket) {
  const objects = await listAllObjects(bucket, "");
  const groups = new Map();
  for (const obj of objects) {
    const parts = obj.key.split("/");
    if (parts.length < 4) continue; // malformed/stray key, skip
    const [username, mediaId, module, runNumStr] = parts;
    const runNum = parseInt(runNumStr, 10);
    if (Number.isNaN(runNum)) continue;
    const groupKey = `${username} ${mediaId} ${module}`;
    let group = groups.get(groupKey);
    if (!group) {
      group = { username, mediaId, module, highestRunNum: runNum, runs: new Set() };
      groups.set(groupKey, group);
    }
    group.runs.add(runNum);
    if (runNum > group.highestRunNum) group.highestRunNum = runNum;
  }
  const rows = Array.from(groups.values()).map((g) => ({
    ...g,
    runs: Array.from(g.runs).sort((a, b) => a - b),
  }));
  rows.sort(
    (a, b) =>
      a.username.localeCompare(b.username) ||
      a.mediaId.localeCompare(b.mediaId) ||
      a.module.localeCompare(b.module)
  );
  return rows;
}
