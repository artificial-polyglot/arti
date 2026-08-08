// output_details.js
// Builds the fixed-row "Details" table for one arti-output run:
// {username}/{media_id}/{module}/{run_num}/{file_type}/{file_content}
//
// Confirmed against courier/courier.go (PersistToBucket/createKey):
//   - run_num is zero-padded to 5 digits ("%05d"), e.g. "00012" - callers of
//     getOutputDetailRows may pass a plain int/string; it gets padded here.
//   - file_type is always one of: request, log, database, output, runtime,
//     duration, status. status is only uploaded when a run errors out (see
//     courier.go), so it won't appear for every run. "Output1..Outputn" is NOT a real subfolder scheme - every
//     output artifact (csv/json/html report, etc.) is uploaded under the same
//     "output/" file_type, distinguished only by filename, so this Worker
//     numbers them Output1..OutputN itself (sorted by filename for a stable
//     order). "database" can likewise hold more than one file (one per DB the
//     run persisted), so it's numbered the same way when there's more than one.
//   - Runtime/Duration are NOT file contents. uploadString() is called with
//     the runtime/duration string itself as the *filename* argument and an
//     empty body, so the value ends up as the last path segment of the key,
//     e.g. ".../runtime/Mon Jul 27 2026 03:04:05 pm MST". There's nothing to
//     fetch: the "content" already extracted from the key IS the display value.
import { listAllObjects } from "./r2_list.js";

// The run-level prefix shared by every object belonging to one run - request,
// output, log, database, status, and (see viewer.js) the notes object all
// live directly under this.
export function outputRunPrefix(username, mediaId, module, runNum) {
  const paddedRunNum = String(runNum).padStart(5, "0");
  return `${username}/${mediaId}/${module}/${paddedRunNum}/`;
}

export async function getOutputDetailRows(bucket, username, mediaId, module, runNum) {
  const prefix = outputRunPrefix(username, mediaId, module, runNum);
  const objects = await listAllObjects(bucket, prefix);

  // Group by file_type (first path segment after the prefix); each group can
  // hold more than one file (output, database).
  const byType = new Map();
  for (const obj of objects) {
    const rest = obj.key.slice(prefix.length);
    const segments = rest.split("/");
    const fileType = segments.shift();
    const content = segments.join("/");
    if (!fileType || !content) continue;
    if (!byType.has(fileType)) byType.set(fileType, []);
    byType.get(fileType).push({ key: obj.key, content });
  }
  for (const list of byType.values()) list.sort((a, b) => a.content.localeCompare(b.content));

  const first = (fileType) => (byType.get(fileType) || [])[0] || null;
  const stringValue = (fileType) => first(fileType)?.content ?? "";

  const rows = [];
  rows.push({ label: "Runtime", value: stringValue("runtime"), viewMode: null, viewKey: null, downloadKey: null });
  rows.push({ label: "Duration", value: stringValue("duration"), viewMode: null, viewKey: null, downloadKey: null });
  rows.push(fileRow("Request", first("request"), true));
  for (const row of numberedRows("Output", byType.get("output") || [], true)) rows.push(row);
  rows.push(fileRow("Log file", first("log"), true));
  rows.push(fileRow("Status", first("status"), true));
  const databases = byType.get("database") || [];
  if (databases.length > 1) {
    for (const row of numberedRows("Database", databases, false)) rows.push(row);
  } else {
    rows.push(fileRow("Database", databases[0] || null, false));
  }
  return rows;
}

// Turns a list of same-file_type objects into Output1../Database1.. rows.
// A single file keeps the "1" suffix (Output1, not bare "Output") so the
// numbering stays consistent whether a run produced one file or several -
// the spec calls this out explicitly ("Output1..Outputn"). Database is the
// opposite case: the spec shows one bare "Database" row, so it only gets
// numbered here if a run actually produced more than one.
function numberedRows(label, list, viewable) {
  return list.map((obj, i) => fileRow(`${label}${i + 1}`, obj, viewable));
}

// Request/Output/Log entries are always some kind of text (yaml, csv, json,
// log, html report, ...) so - unlike the Input tab's classifyView, which
// defaults to no view action at all - these default to a plain-text "show"
// preview, only switching to "open" (render as a real page in a new tab)
// for actual .html/.htm reports.
function viewModeFor(filename) {
  const lower = filename.toLowerCase();
  return lower.endsWith(".html") || lower.endsWith(".htm") ? "open" : "show";
}

function fileRow(label, obj, viewable) {
  if (!obj) return { label, value: "", viewMode: null, viewKey: null, downloadKey: null };
  return {
    label,
    value: obj.content,
    viewMode: viewable ? viewModeFor(obj.content) : null,
    viewKey: viewable ? obj.key : null,
    downloadKey: obj.key,
  };
}
