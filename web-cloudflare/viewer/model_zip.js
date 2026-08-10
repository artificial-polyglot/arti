// model_zip.js
// Bundles one arti-models entry (modelType/langIso/...) into a zip archive
// for download. R2 lays each model out as a tensor file plus a
// processor_<langIso>/ directory of tokenizer files, both directly under
// modelType/langIso/ - this renames that prefix to just langIso/ so
// unzipping drops in the folder a local model directory expects, with the
// tensor file and the tokenizer directory both inside it.

import { listAllObjects, padRunNum } from "./r2_list.js";
import { buildZip } from "./zip.js";

export async function buildModelZip(bucket, modelType, langIso, runNum) {
  const prefix = `${modelType}/${langIso}/${padRunNum(runNum)}/`;
  const objects = await listAllObjects(bucket, prefix);
  if (objects.length === 0) return null;

  const entries = [];
  for (const obj of objects) {
    const relPath = obj.key.slice(prefix.length);
    if (!relPath) continue; // directory placeholder object, not a real file
    const body = await bucket.get(obj.key);
    if (!body) continue;
    entries.push({
      name: `${langIso}/${relPath}`,
      data: new Uint8Array(await body.arrayBuffer()),
      date: obj.uploaded,
    });
  }
  return buildZip(entries);
}
