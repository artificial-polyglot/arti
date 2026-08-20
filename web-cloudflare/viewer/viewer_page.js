// viewer_page.js
// Self-contained HTML/CSS/JS for the viewer page. Kept as a plain exported
// string (rather than a static-assets directory) so the whole Worker stays a
// couple of importable files, matching web-cloudflare/upload's single-file
// style. No build step, no external scripts - just fetch() against this same
// Worker's /api/* routes and a small hand-rolled sortable table.

export const PAGE_HTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Arti R2 Viewer</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: system-ui, sans-serif; margin: 0; display: flex; height: 100vh; }
  nav { display: flex; flex-direction: column; gap: 8px; padding: 16px; width: 140px;
        border-right: 1px solid rgba(128,128,128,0.35); flex-shrink: 0; }
  nav button { padding: 10px; font-size: 1rem; cursor: pointer; border-radius: 6px;
               border: 1px solid rgba(128,128,128,0.35); background: transparent; color: inherit; }
  nav button.active { font-weight: 600; border-color: currentColor; }
  main { flex: 1; padding: 16px 24px; overflow: auto; }
  table { border-collapse: collapse; width: 100%; }
  th, td { text-align: left; padding: 6px 12px; border-bottom: 1px solid rgba(128,128,128,0.25); }
  th { cursor: pointer; user-select: none; white-space: nowrap; }
  th.sortable:hover { text-decoration: underline; }
  th .arrow { opacity: 0.6; font-size: 0.8em; }
  button.action { padding: 3px 10px; cursor: pointer; }
  input[type=number] { width: 4em; }
  .crumbs { margin-bottom: 12px; }
  .crumbs a { cursor: pointer; text-decoration: underline; }
  dialog { max-width: 90vw; max-height: 85vh; overflow: auto; }
  dialog pre { white-space: pre-wrap; word-break: break-word; }
  .empty { opacity: 0.7; font-style: italic; }
  .file-preview { white-space: pre-wrap; word-break: break-word; border: 1px solid rgba(128,128,128,0.25);
                   border-radius: 6px; padding: 8px 12px; max-height: 400px; overflow: auto; }
  .title-row { display: flex; align-items: center; gap: 10px; }
  .notes-edit { width: 100%; min-height: 150px; box-sizing: border-box; font: inherit;
                 padding: 8px 12px; border-radius: 6px; border: 1px solid rgba(128,128,128,0.35); }
</style>
</head>
<body>
<nav>
  <button data-view="output">Output</button>
  <button data-view="input">Input</button>
  <button data-view="models">Models</button>
</nav>
<main id="main"></main>
<dialog id="dlg"><button id="dlgClose" style="float:right">Close</button><div id="dlgBody"></div></dialog>
<script>
(function () {
  const main = document.getElementById("main");
  const dlg = document.getElementById("dlg");
  document.getElementById("dlgClose").onclick = () => dlg.close();
  // Stop any playing audio no matter how the dialog closed (Close button,
  // Esc key, or backdrop click via dlg.close()/requestClose()).
  dlg.addEventListener("close", () => {
    const audio = document.getElementById("dlgBody").querySelector("audio");
    if (audio) { audio.pause(); audio.removeAttribute("src"); audio.load(); }
  });

  const navButtons = Array.from(document.querySelectorAll("nav button"));
  navButtons.forEach((btn) => btn.addEventListener("click", () => showTopLevel(btn.dataset.view)));

  function setActive(view) {
    navButtons.forEach((b) => b.classList.toggle("active", b.dataset.view === view));
  }

  async function getJSON(url) {
    const res = await fetch(url);
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
    return res.json();
  }

  function fileUrl(bucket, key, mode, tail) {
    return "/file?bucket=" + encodeURIComponent(bucket) + "&key=" + encodeURIComponent(key) + "&mode=" + mode +
      (tail ? "&tail=1" : "");
  }

  function showText(title, text) {
    document.getElementById("dlgBody").innerHTML = "";
    const h = document.createElement("h3"); h.textContent = title;
    const pre = document.createElement("pre"); pre.textContent = text;
    document.getElementById("dlgBody").append(h, pre);
    dlg.showModal();
  }

  function showAudio(title, src) {
    document.getElementById("dlgBody").innerHTML = "";
    const h = document.createElement("h3"); h.textContent = title;
    const audio = document.createElement("audio");
    audio.controls = true; audio.src = src; audio.style.width = "100%";
    document.getElementById("dlgBody").append(h, audio);
    dlg.showModal();
  }

  async function runAction(action, bucket, key, filename, tail, viewMode) {
    if (action === "download") {
      if (viewMode === "open") {
        await downloadSignedReport(bucket, key);
      } else {
        window.location = fileUrl(bucket, key, "download");
      }
    } else if (action === "show") {
      const res = await fetch(fileUrl(bucket, key, "show", tail));
      showText(filename, await res.text());
    } else if (action === "open") {
      // Real .html/.htm pages get their own tab so the browser renders them
      // normally (styles/scripts intact), rather than the plain-text dialog.
      window.open(fileUrl(bucket, key, "open"), "_blank");
    } else if (action === "play") {
      showAudio(filename, fileUrl(bucket, key, "play"));
    }
  }

  // A real .html/.htm report (viewMode "open") embeds audio links that are
  // either relative or already-expired signed URLs from generation time -
  // fine when viewed live through this Worker, dead once the file leaves it.
  // audio_file_urls.json (generic.OutputAudioFiles) sits right next to the
  // report under the same run prefix, one filename swap away from the
  // report's own key, so fetch it, ask /api/sign-audio-urls to fill in fresh
  // signed URLs, and swap them into a copy of the report text before handing
  // it to the browser as a download. Reports with no manifest (older runs,
  // or ones with no audio) just download unchanged.
  async function downloadSignedReport(bucket, key) {
    const htmlRes = await fetch(fileUrl(bucket, key, "open"));
    let html = await htmlRes.text();
    const manifestKey = key.slice(0, key.lastIndexOf("/") + 1) + "audio_file_urls.json";
    const signRes = await fetch(
      "/api/sign-audio-urls?bucket=" + encodeURIComponent(bucket) + "&key=" + encodeURIComponent(manifestKey),
    );
    if (signRes.ok) {
      const entries = await signRes.json();
      for (const entry of entries) {
        if (entry.unsigned_url && entry.signed_url) html = html.replaceAll(entry.unsigned_url, entry.signed_url);
      }
    }
    const blobUrl = URL.createObjectURL(new Blob([html], { type: "text/html" }));
    const a = document.createElement("a");
    a.href = blobUrl;
    a.download = key.split("/").pop();
    document.body.append(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(blobUrl);
  }

  // --- generic sortable table -------------------------------------------
  // columns: [{ key, label, sortable=true, render(row) -> Node|string }]
  function renderTable(container, columns, rows, defaultSortKeys) {
    container.innerHTML = "";
    if (rows.length === 0) {
      const p = document.createElement("p");
      p.className = "empty";
      p.textContent = "No data.";
      container.append(p);
      return;
    }
    let sortKey = defaultSortKeys[0];
    let sortDir = 1;

    const table = document.createElement("table");
    const thead = document.createElement("thead");
    const headRow = document.createElement("tr");
    columns.forEach((col) => {
      const th = document.createElement("th");
      th.textContent = col.label;
      if (col.sortable !== false) {
        th.classList.add("sortable");
        th.addEventListener("click", () => {
          if (sortKey === col.key) sortDir = -sortDir;
          else { sortKey = col.key; sortDir = 1; }
          draw();
        });
      }
      headRow.append(th);
    });
    thead.append(headRow);
    const tbody = document.createElement("tbody");
    table.append(thead, tbody);
    container.append(table);

    function draw() {
      const sorted = rows.slice().sort((a, b) => {
        const av = a[sortKey], bv = b[sortKey];
        if (av === bv) return 0;
        const cmp = typeof av === "number" && typeof bv === "number" ? av - bv : String(av).localeCompare(String(bv));
        return cmp * sortDir;
      });
      tbody.innerHTML = "";
      sorted.forEach((row) => {
        const tr = document.createElement("tr");
        columns.forEach((col) => {
          const td = document.createElement("td");
          const rendered = col.render ? col.render(row) : row[col.key];
          if (rendered instanceof Node) td.append(rendered);
          else td.textContent = rendered ?? "";
          tr.append(td);
        });
        tbody.append(tr);
      });
      headRow.querySelectorAll("th").forEach((th, i) => {
        const arrow = th.querySelector(".arrow");
        if (arrow) arrow.remove();
        if (columns[i].key === sortKey) {
          const span = document.createElement("span");
          span.className = "arrow";
          span.textContent = sortDir === 1 ? " ▲" : " ▼";
          th.append(span);
        }
      });
    }
    draw();
  }

  function actionButton(label, onClick) {
    const btn = document.createElement("button");
    btn.className = "action";
    btn.textContent = label;
    btn.addEventListener("click", onClick);
    return btn;
  }

  function fmtDate(iso) {
    if (!iso) return "";
    const d = new Date(iso);
    return isNaN(d) ? String(iso) : d.toLocaleString();
  }

  // --- views ---------------------------------------------------------------
  async function showTopLevel(view) {
    setActive(view);
    main.innerHTML = "<p>Loading…</p>";
    if (view === "models") return showModels();
    if (view === "input") return showInputList();
    if (view === "output") return showOutputList();
  }

  async function showModels() {
    const rows = await getJSON("/api/models");
    main.innerHTML = "<h2>Models</h2>";
    const div = document.createElement("div");
    main.append(div);
    renderTable(div, [
      {
        key: "details", label: "", sortable: false,
        render: (r) => actionButton("Details", () => {
          const runNum = r._modelRunInput ? r._modelRunInput.value : r.highestRunNum;
          showModelDetails(r.modelType, r.langIso, runNum);
        }),
      },
      {
        key: "download", label: "", sortable: false,
        render: (r) => actionButton("Download", () => {
          const runNum = r._modelRunInput ? r._modelRunInput.value : r.highestRunNum;
          window.location = "/api/models/download?modelType=" + encodeURIComponent(r.modelType) +
            "&langIso=" + encodeURIComponent(r.langIso) + "&runNum=" + encodeURIComponent(runNum);
        }),
      },
      {
        key: "highestRunNum", label: "Model#", render: (r) => {
          const input = document.createElement("input");
          input.type = "number"; input.min = 1; input.max = r.highestRunNum; input.value = r.highestRunNum;
          input.addEventListener("input", () => {
            const uploaded = r.runs[input.value];
            if (uploaded && r._updatedSpan) r._updatedSpan.textContent = fmtDate(uploaded);
          });
          r._modelRunInput = input;
          return input;
        },
      },
      { key: "langIso", label: "Lang ISO" },
      {
        key: "uploaded", label: "Updated", render: (r) => {
          const span = document.createElement("span");
          span.textContent = fmtDate(r.uploaded);
          r._updatedSpan = span;
          return span;
        },
      },
    ], rows, ["langIso"]);
  }

  async function showModelDetails(modelType, langIso, runNum) {
    const qs = new URLSearchParams({ modelType, langIso, runNum }).toString();
    const rows = await getJSON("/api/models/details?" + qs);
    main.innerHTML =
      '<div class="crumbs"><a id="back">← Models</a></div><h2>Model: ' +
      [modelType, langIso, "run " + runNum].map(escapeHtml).join(" / ") + "</h2>";
    document.getElementById("back").addEventListener("click", showModels);
    const div = document.createElement("div");
    main.append(div);
    renderTable(div, [
      {
        key: "show", label: "", sortable: false,
        render: (r) => actionButton("Show", () => runAction("show", "models", r.key, r.filename)),
      },
      {
        key: "download", label: "", sortable: false,
        render: (r) => actionButton("Download", () => runAction("download", "models", r.key, r.filename)),
      },
      { key: "filename", label: "Filename" },
      { key: "uploaded", label: "Uploaded", render: (r) => fmtDate(r.uploaded) },
    ], rows, ["filename"]);
  }

  async function showInputList() {
    const rows = await getJSON("/api/input");
    main.innerHTML = "<h2>Input</h2>";
    const div = document.createElement("div");
    main.append(div);
    renderTable(div, [
      { key: "details", label: "", sortable: false, render: (r) => actionButton("Details", () => showInputDetails(r.mediaId)) },
      { key: "mediaId", label: "Media ID" },
      { key: "uploaded", label: "Uploaded", render: (r) => fmtDate(r.uploaded) },
    ], rows, ["mediaId"]);
  }

  async function showInputDetails(mediaId) {
    const rows = await getJSON("/api/input/" + encodeURIComponent(mediaId));
    main.innerHTML =
      '<div class="crumbs"><a id="back">← Input</a></div><h2>Input: ' + escapeHtml(mediaId) + "</h2>";
    document.getElementById("back").addEventListener("click", showInputList);
    const div = document.createElement("div");
    main.append(div);
    const actionLabel = { play: "Play", show: "View", open: "Open" };
    renderTable(div, [
      {
        key: "action", label: "", sortable: false,
        render: (r) => (r.action ? actionButton(actionLabel[r.action], () => runAction(r.action, "input", r.key, r.filename)) : ""),
      },
      {
        key: "download", label: "", sortable: false,
        render: (r) => actionButton("Download", () => runAction("download", "input", r.key, r.filename)),
      },
      { key: "prefix", label: "Prefix" },
      { key: "filename", label: "Filename" },
      { key: "uploaded", label: "Uploaded", render: (r) => fmtDate(r.uploaded) },
    ], rows, ["prefix", "filename"]);
  }

  async function showOutputList() {
    const rows = await getJSON("/api/output");
    main.innerHTML = "<h2>Output</h2>";
    const div = document.createElement("div");
    main.append(div);
    renderTable(div, [
      {
        key: "details", label: "", sortable: false,
        render: (r) => actionButton("Details", () => {
          const runNum = r._runNumInput ? r._runNumInput.value : r.highestRunNum;
          showOutputDetails(r.username, r.mediaId, r.module, runNum);
        }),
      },
      {
        key: "highestRunNum", label: "Run #", render: (r) => {
          const input = document.createElement("input");
          input.type = "number"; input.min = 1; input.max = r.highestRunNum; input.value = r.highestRunNum;
          r._runNumInput = input;
          return input;
        },
      },
      { key: "username", label: "Username" },
      { key: "mediaId", label: "Media ID" },
      { key: "module", label: "Module" },
    ], rows, ["username", "mediaId"]);
  }

  async function showOutputDetails(username, mediaId, module, runNum) {
    const qs = new URLSearchParams({ username, mediaId, module, runNum }).toString();
    const rows = await getJSON("/api/output/details?" + qs);
    main.innerHTML =
      '<div class="crumbs"><a id="back">← Output</a></div><h2>Output: ' +
      [username, mediaId, module, "run " + runNum].map(escapeHtml).join(" / ") + "</h2>";
    document.getElementById("back").addEventListener("click", showOutputList);

    const viewLabel = { show: "Show", open: "Open" };
    const table = document.createElement("table");
    const tbody = document.createElement("tbody");
    let requestRow = null;
    let statusRow = null;
    rows.forEach((row) => {
      if (row.label === "Request") { requestRow = row; return; }
      if (row.label === "Status") { statusRow = row; return; }
      const tr = document.createElement("tr");
      const viewTd = document.createElement("td");
      if (row.viewMode) {
        const tail = row.label === "Log file";
        viewTd.append(actionButton(viewLabel[row.viewMode], () => runAction(row.viewMode, "output", row.viewKey, row.label, tail)));
      }
      const downloadTd = document.createElement("td");
      if (row.downloadKey) {
        downloadTd.append(actionButton("Download", () => runAction("download", "output", row.downloadKey, row.label, false, row.viewMode)));
      } else {
        downloadTd.textContent = row.label;
      }
      const valueTd = document.createElement("td"); valueTd.textContent = row.value;
      tr.append(viewTd, downloadTd, valueTd);
      tbody.append(tr);
    });
    table.append(tbody);
    main.append(table);

    const notesKey = username + "/" + mediaId + "/" + module + "/" + String(runNum).padStart(5, "0") + "/notes";
    const notesTitle = document.createElement("h3"); notesTitle.className = "title-row";
    const notesTitleText = document.createElement("span"); notesTitleText.textContent = "Notes";
    notesTitle.append(notesTitleText, actionButton("Edit", () => editNotes()));
    const notesBlock = document.createElement("div"); notesBlock.className = "file-preview";
    main.append(notesTitle, notesBlock);

    let notesText = "";
    const notesRes = await fetch(fileUrl("output", notesKey, "show"));
    if (notesRes.ok) notesText = await notesRes.text();
    notesBlock.textContent = notesText;

    function viewNotes() {
      notesBlock.className = "file-preview";
      notesBlock.innerHTML = "";
      notesBlock.textContent = notesText;
    }

    function editNotes() {
      notesBlock.className = "";
      notesBlock.innerHTML = "";
      const textarea = document.createElement("textarea");
      textarea.className = "notes-edit";
      textarea.value = notesText;
      const btnRow = document.createElement("div");
      btnRow.style.marginTop = "8px";
      btnRow.append(
        actionButton("Save", async () => {
          notesText = textarea.value;
          await fetch("/api/output/notes", {
            method: "POST",
            headers: { "content-type": "application/json" },
            body: JSON.stringify({ username, mediaId, module, runNum, text: notesText }),
          });
          viewNotes();
        }),
        actionButton("Cancel", viewNotes),
      );
      notesBlock.append(textarea, btnRow);
      textarea.focus();
    }

    const statusH3 = document.createElement("h3"); statusH3.textContent = "Status";
    const statusPre = document.createElement("pre"); statusPre.className = "file-preview";
    main.append(statusH3, statusPre);
    if (statusRow && statusRow.viewKey) {
      const res = await fetch(fileUrl("output", statusRow.viewKey, "show"));
      const text = await res.text();
      statusPre.textContent = text.trim() ? text : "OK";
    } else {
      statusPre.textContent = "OK";
    }

    if (requestRow && requestRow.viewKey) {
      const h3 = document.createElement("h3"); h3.className = "title-row";
      const h3Text = document.createElement("span"); h3Text.textContent = "Request";
      h3.append(h3Text, actionButton("Download", () => runAction("download", "output", requestRow.downloadKey, requestRow.label)));
      const pre = document.createElement("pre"); pre.className = "file-preview";
      main.append(h3, pre);
      const res = await fetch(fileUrl("output", requestRow.viewKey, "show"));
      pre.textContent = await res.text();
    }
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  }

  showTopLevel("output");
})();
</script>
</body>
</html>
`;
