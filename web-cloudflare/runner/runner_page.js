// runner_page.js
// Self-contained HTML/CSS/JS for the job-request form. Kept as a plain
// exported string (matching web-cloudflare/viewer's style) so the whole
// Worker stays a couple of importable files, no build step.
//
// The form markup/styling below is carried over as-is from
// arti/controller/web/arti.html, with two exceptions the user asked to drop:
//   - the AWS-credentials dropzone (only meaningful for the old S3-upload
//     client, which isn't used here)
//   - the Upload button, YAML file-load input, and validation error modal
//     (all dead weight without arti-main.js/arti-upload.js/arti-validation.js)
// The folder dropzone markup is kept for a later pass but isn't wired to any
// JS here.
//
// Everything under the <script> tags is new: it reads the form fields into
// one flat dict and JSON.stringifies it, hands that JSON to the
// request/validate WASM module (validate.wasm, built from
// web-cloudflare/validate_wasm) for the same validation the server does.
// On failure, the returned errors are listed at the top of the page; on
// success, the "Save JSON" button downloads the generated JSON as before.
// "Save JSON" stands in for a future "Submit Job" button - for now all
// operations (validate, then save) are tied to it.

export const PAGE_HTML = `<!doctype html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Artie Media and Job Uploader</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background-color: #2b2b2b;
            color: #ffffff;
            padding: 20px;
            line-height: 1.5;
        }

        .container {
            max-width: 800px;
            margin: 0 auto;
            background-color: #3a3a3a;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
            position: relative;
        }

        .folder-dropzone {
            flex-shrink: 0;
            width: 200px;
            height: 80px;
            border: 2px dashed #666;
            border-radius: 8px;
            background-color: #4a4a4a;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            transition: all 0.3s ease;
            font-size: 12px;
            text-align: center;
            padding: 10px;
        }

        .folder-dropzone>* {
            pointer-events: none;
        }

        .folder-status {
            pointer-events: auto !important;
        }

        .folder-dropzone:hover {
            border-color: #0066cc;
            background-color: #5a5a5a;
        }

        .folder-dropzone.dragover {
            border-color: #00cc00;
            background-color: #2a4a2a;
        }

        .folder-dropzone.success {
            border-color: #00cc00;
            background-color: #1a4d1a;
        }

        .folder-dropzone.error {
            border-color: #ff4444;
            background-color: #4d1a1a;
        }

        .folder-dropzone.processing {
            border-color: #ffaa00;
            background-color: #4a3a1a;
        }

        .folder-status {
            font-size: 10px;
            color: #ccc;
            margin-top: 5px;
        }

        .folder-status.success {
            color: #90ee90;
        }

        .folder-status.error {
            color: #ffb3b3;
        }

        .folder-status.processing {
            color: #ffaa00;
        }

        .folder-status.default {
            color: #ccc;
        }

        .folder-progress {
            margin-top: 8px;
            width: 100%;
        }

        .folder-progress .progress-bar {
            width: 100%;
            height: 12px;
            background-color: #2a2a2a;
            border-radius: 6px;
            overflow: hidden;
            margin-bottom: 5px;
        }

        .folder-progress .progress-fill {
            height: 100%;
            background-color: #ffaa00;
            transition: width 0.3s ease;
        }

        .folder-progress .progress-text {
            color: #ffaa00;
            font-size: 9px;
            text-align: center;
        }

        .header {
            display: flex;
            align-items: center;
            margin-bottom: 30px;
            padding: 20px 0;
            gap: 20px;
        }

        .logo {
            max-width: 300px;
            height: auto;
            border-radius: 8px;
            flex-shrink: 0;
        }

        .subtitle {
            color: #ffffff;
            font-size: 32px;
            font-weight: 400;
            margin: 0;
            flex: 1;
            text-align: center;
        }

        .button-group {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 10px;
            margin-bottom: 20px;
            flex-wrap: wrap;
        }

        button {
            background-color: #4a4a4a;
            color: #ffffff;
            border: 1px solid #666;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.2s;
        }

        button:hover {
            background-color: #5a5a5a;
        }

        button:active {
            background-color: #3a3a3a;
        }

        button.primary {
            background-color: #0066cc;
            border-color: #0052a3;
        }

        button.primary:hover {
            background-color: #0052a3;
        }

        .separator {
            height: 1px;
            background-color: #666;
            margin: 20px 0;
        }

        .form-group {
            margin-bottom: 20px;
        }

        .form-row {
            display: flex;
            align-items: center;
            gap: 15px;
            margin-bottom: 10px;
            flex-wrap: wrap;
        }

        label {
            min-width: 120px;
            font-weight: 500;
            color: #ffffff;
        }

        input[type="text"],
        input[type="number"],
        textarea,
        select {
            flex: 1;
            min-width: 200px;
            background-color: #4a4a4a;
            color: #ffffff;
            border: 1px solid #666;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 14px;
            font-family: inherit;
        }

        textarea {
            resize: none;
            overflow: hidden;
        }

        select {
            cursor: pointer;
        }

        input[type="text"]:focus,
        input[type="number"]:focus,
        textarea:focus,
        select:focus {
            outline: none;
            border-color: #0066cc;
            box-shadow: 0 0 0 2px rgba(0, 102, 204, 0.2);
        }

        input[type="text"]::placeholder,
        textarea::placeholder {
            color: #999;
        }

        input[type="text"].optional,
        textarea.optional {
            background-color: #3a3a3a;
            border: 1px solid #555;
            color: #ccc;
        }

        input[type="text"].optional::placeholder,
        textarea.optional::placeholder {
            color: #777;
            font-style: italic;
        }

        input[type="text"].optional:focus,
        textarea.optional:focus {
            background-color: #4a4a4a;
            border-color: #666;
        }

        input[type="text"].required-error {
            border-color: #ff4444;
            box-shadow: 0 0 0 2px rgba(255, 68, 68, 0.2);
        }

        .radio-group,
        .checkbox-group {
            margin-bottom: 15px;
        }

        .radio-group label {
            min-width: auto;
            margin-right: 15px;
            font-weight: normal;
        }

        .radio-options {
            display: flex;
            flex-direction: column;
            gap: 8px;
            margin-left: 135px;
        }

        .radio-option {
            display: flex;
            align-items: flex-start;
            gap: 8px;
        }

        .radio-option input[type="radio"] {
            margin-top: 2px;
        }

        .radio-option label {
            font-weight: normal;
            min-width: auto;
        }

        .checkbox-option {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 10px;
        }

        .checkbox-option input[type="checkbox"] {
            margin: 0;
        }

        .description {
            color: #ccc;
            font-size: 12px;
            margin-left: 8px;
        }

        .status {
            margin-top: 20px;
            padding: 10px;
            border-radius: 4px;
            display: none;
        }

        .status.success {
            background-color: #1a4d1a;
            border: 1px solid #2d6b2d;
            color: #90ee90;
        }

        .status.error {
            background-color: #4d1a1a;
            border: 1px solid #6b2d2d;
            color: #ffb3b3;
        }

        .status.warning {
            background-color: #4d3a1a;
            border: 1px solid #6b4d2d;
            color: #ffd700;
        }

        .error-banner {
            display: none;
            margin-bottom: 20px;
            padding: 10px 15px;
            border-radius: 4px;
            background-color: #4d1a1a;
            border: 1px solid #6b2d2d;
            color: #ffb3b3;
        }

        .error-banner div {
            padding: 2px 0;
        }

        /* Required field validation styling */
        .required-error {
            border: 1px solid #cc6666 !important;
            background-color: #3d2a2a !important;
        }

        .required-valid {
            border: 1px solid #66cc66 !important;
            background-color: #2a3d2a !important;
        }

        @media (max-width: 600px) {
            .header {
                flex-direction: column;
                text-align: center;
                gap: 10px;
            }

            .logo {
                max-width: 200px;
            }

            .subtitle {
                font-size: 24px;
            }

            .form-row {
                flex-direction: column;
                align-items: flex-start;
            }

            .radio-options {
                margin-left: 0;
            }

            label {
                min-width: auto;
            }
        }
    </style>
</head>

<body>
    <div class="container">
        <div class="error-banner" id="errorBanner"></div>

        <div class="header">
            <img src="/arti.jpg" alt="Artie Logo" class="logo">
            <h2 class="subtitle">Arti2<br>Task and Media<br>Uploader</h2>
        </div>

        <div class="button-group">
            <button onclick="clearForm()">Clear</button>
            <button onclick="saveJSON()">Save JSON</button>
            <div class="folder-dropzone" id="folderDropzone">
                <div>&#128193; Drag a media folder or YAML file here, or press to select</div>
                <div class="folder-status" id="folderStatus">No folder or file selected</div>
                <div class="folder-progress" id="folderProgress" style="display: none;"></div>
            </div>
        </div>

        <div class="separator"></div>

        <form id="requestForm">
            <div class="form-group">
                <div class="form-row">
                    <label for="username">username:</label>
                    <input type="text" id="username" style="max-width: 400px;"
                        placeholder="Enter a unique username for yourself (used to group datasets on the server)."
                        required>
                </div>
            </div>

            <div class="form-group">
                <div class="form-row">
                    <label for="datasetName">dataset_name:</label>
                    <input type="text" id="datasetName" style="max-width: 400px;"
                        placeholder="Enter a unique name for this dataset (unique within your username that is).">
                </div>
            </div>

            <div class="form-group">
                <div class="form-row" style="align-items: flex-start;">
                    <label for="textData" style="padding-top: 6px;">text_data:</label>
                    <textarea id="textData" rows="2"
                        placeholder="s3://bucket-name/path/to/files/*.sfm"></textarea>
                </div>
            </div>

            <div class="form-group">
                <div class="radio-group">
                    <label>text_format:</label>
                    <div class="radio-options" style="flex-direction: row; gap: 20px; align-items: center;">
                        <div class="radio-option">
                            <input type="radio" id="text_format_sfm" name="text_format" value="sfm" checked>
                            <label for="text_format_sfm">SFM (.sfm)</label>
                        </div>
                        <div class="radio-option">
                            <input type="radio" id="text_format_usx" name="text_format" value="usx">
                            <label for="text_format_usx">USX (.usx)</label>
                        </div>
                    </div>
                </div>
            </div>

            <div class="form-group">
                <div class="form-row" style="align-items: flex-start;">
                    <label for="audioData" style="padding-top: 6px;">audio_data:</label>
                    <textarea id="audioData" rows="2"
                        placeholder="s3://bucket-name/path/to/files/*.mp3"></textarea>
                </div>
            </div>

            <div class="form-group">
                <div class="form-row">
                    <label for="languageIso">language_iso:</label>
                    <input type="text" id="languageIso"
                        style="flex: none; width: 90px; min-width: 0;"
                        placeholder="eng"
                        maxlength="3">
                </div>
            </div>

            <div class="form-group">
                <div class="form-row">
                    <label for="altLanguage">alt_language: <span
                            style="color: #888; font-size: 12px;">(optional)</span></label>
                    <input type="text" id="altLanguage" class="optional"
                        placeholder="Enter a different ISO code, to override the above.">
                </div>
            </div>

            <div class="form-group">
                <div class="form-group">
                    <div class="radio-group">
                        <label>model:</label>
                        <div class="radio-options">
                            <div class="radio-option">
                                <input type="radio" id="training_mms_adapter" name="training" value="mms_adapter"
                                    checked>
                                <label for="training_mms_adapter">mms_adapter - use a locally-trained model for this
                                    iso</label>
                                <div style="margin-left: 24px; margin-top: 4px;">
                                    <input type="checkbox" id="redoTraining" style="width: auto; margin-right: 5px;">
                                    <label for="redoTraining" style="font-weight: normal;">Redo Training</label>
                                </div>
                            </div>
                            <div class="radio-option">
                                <input type="radio" id="training_no_training" name="training" value="no_training">
                                <label for="training_no_training">mms - use mms' model for this iso</label>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <div class="radio-group">
                        <label>timestamps:</label>
                        <div class="radio-options">
                            <div class="radio-option">
                                <input type="radio" id="timestamps_mms_align" name="timestamps" value="mms_align"
                                    checked>
                                <label for="timestamps_mms_align">mms_align - fastest and most accurate, but can fail
                                    due to
                                    insufficient memory</label>
                            </div>
                            <div class="radio-option">
                                <input type="radio" id="timestamps_mms_fa_verse" name="timestamps" value="mms_fa_verse">
                                <label for="timestamps_mms_fa_verse">mms_fa_verse - slower and less accurate, but try
                                    this
                                    if mms_align fails</label>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <div class="checkbox-group">
                        <div class="checkbox-option">
                            <input type="checkbox" id="compare" checked>
                            <label for="compare">compare: report comparing audio transcript to correct text</label>
                        </div>
                        <div class="checkbox-option">
                            <input type="checkbox" id="proofing">
                            <label for="proofing">proofing: report highlighting probable incorrect words</label>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <div class="form-row">
                        <label for="gordonFilter">gordon_filter:</label>
                        <input type="number" id="gordonFilter" style="flex: none; width: 90px; min-width: 0;"
                            placeholder="0" value="0">
                    </div>
                </div>

                <div class="form-group">
                    <div class="form-row">
                        <label for="notifyOk">notify_ok: <span style="color: #888; font-size: 12px;">(comma-separated
                                emails)</span></label>
                        <input type="text" id="notifyOk"
                            value="ntfy/arti2"
                            placeholder="ewallace@fcbhmail.org, gfiddes@fcbhmail.org, edomschot@fcbhmail.org">
                    </div>
                </div>

                <div class="form-group">
                    <div class="form-row">
                        <label for="notifyErr">notify_err: <span style="color: #888; font-size: 12px;">(comma-separated
                                emails)</span></label>
                        <input type="text" id="notifyErr"
                            value="ntfy/arti2"
                            placeholder="gary@shortsands.com, ewallace@fcbhmail.org">
                    </div>
                </div>
            </div>
        </form>

        <div class="status" id="status"></div>
    </div>

    <!--
    wasm_exec.js must come before our inline script below, as it defines
    the Go class used to load and run validate.wasm.
    -->
    <script src="/wasm_exec.js"></script>
    <script>
    (function () {
        // ---- load the request/validate WASM module (built from
        // web-cloudflare/validate_wasm) so form validation can run the same
        // Go code the server uses, before the JSON is ever downloaded. -----
        var go = new Go();
        var wasmReady = fetch('/validate.wasm')
            .then(function (resp) { return resp.arrayBuffer(); })
            .then(function (bytes) { return WebAssembly.instantiate(bytes, go.importObject); })
            .then(function (result) { go.run(result.instance); });

        // Relative paths of the files found in the last dropped directory,
        // root directory name included as the prefix (matching what will
        // later be uploaded to the arti-input bucket). Populated by the
        // dropzone's drop handler; validated only when Save JSON is
        // clicked, alongside the form itself - see saveJSON().
        var droppedFilePaths = [];
        /* ---- old defaulting/validation/nested-settings logic: this is now
        done in Go/WASM on the server instead of here in JS. Left in place,
        commented out, for reference while that port happens. --------------

        var DEFAULT_NOTIFY_OK = ['gary@shortsands.com'];
        var DEFAULT_NOTIFY_ERR = ['gary@shortsands.com'];
        var DEFAULT_MMS_ADAPTER = {
            batch_mb: 4,
            num_epochs: 16,
            learning_rate: 0.001,
            warmup_pct: 12,
            grad_norm_max: 0.4
        };
        var DEFAULT_COMPARE_SETTINGS = {
            lower_case: true,
            remove_prompt_chars: true,
            remove_punctuation: true,
            double_quotes: { remove: true },
            apostrophe: { remove: true },
            hyphen: { remove: true },
            diacritical_marks: { normalize_nfc: true }
        };

        function validateLanguageISO(iso) {
            return /^[a-zA-Z]{3}$/.test(iso);
        }

        function validateS3URL(url) {
            return /^s3:\\/\\/[a-zA-Z0-9.-]+\\/.+$/.test(url);
        }

        function parseEmailList(emailString) {
            if (!emailString || !emailString.trim()) return [];
            return emailString.split(',')
                .map(function (email) { return email.trim(); })
                .filter(function (email) { return email.length > 0; });
        }

        function validateEmailList(emailString) {
            if (!emailString || !emailString.trim()) return true;
            var emails = parseEmailList(emailString);
            var emailRegex = /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/;
            return emails.every(function (email) { return emailRegex.test(email); });
        }

        function updateRequiredFieldStyling() {
            var requiredFields = ['datasetName', 'username', 'languageIso', 'textData', 'audioData'];
            requiredFields.forEach(function (fieldId) {
                var field = document.getElementById(fieldId);
                if (!field) return;
                var value = field.value.trim();
                field.classList.remove('required-error', 'required-valid');
                field.classList.add(value.length === 0 ? 'required-error' : 'required-valid');
            });
        }

        function validateForm() {
            var requiredFields = [
                { id: 'datasetName', name: 'Dataset name' },
                { id: 'username', name: 'Username' },
                { id: 'languageIso', name: 'Language ISO' },
                { id: 'textData', name: 'Text data (S3 path)' },
                { id: 'audioData', name: 'Audio data (S3 path)' }
            ];
            var errors = [];

            requiredFields.forEach(function (field) {
                document.getElementById(field.id).classList.remove('required-error');
            });

            requiredFields.forEach(function (field) {
                var element = document.getElementById(field.id);
                var value = element.value.trim();

                if (!value) {
                    errors.push(field.name);
                    element.classList.add('required-error');
                } else if (field.id === 'languageIso' && !validateLanguageISO(value)) {
                    errors.push('Language ISO must be exactly 3 letters (a-z, A-Z)');
                    element.classList.add('required-error');
                } else if (field.id === 'textData' && !validateS3URL(value)) {
                    errors.push('Text data must be a valid S3 URL (s3://bucket/path)');
                    element.classList.add('required-error');
                } else if (field.id === 'audioData' && !validateS3URL(value)) {
                    errors.push('Audio data must be a valid S3 URL (s3://bucket/path)');
                    element.classList.add('required-error');
                }
            });

            var notifyOkValue = document.getElementById('notifyOk').value.trim();
            if (!validateEmailList(notifyOkValue)) errors.push('Notify OK must contain valid email addresses (comma-separated)');
            var notifyErrValue = document.getElementById('notifyErr').value.trim();
            if (!validateEmailList(notifyErrValue)) errors.push('Notify Error must contain valid email addresses (comma-separated)');

            if (errors.length > 0) {
                showStatus('Validation errors: ' + errors.join(', '), 'error');
                return false;
            }
            return true;
        }

        // ---- pull the form fields into a Map (with nested Maps for grouped
        // settings), then flatten that Map tree into plain objects/arrays so
        // JSON.stringify can serialize it. ----------------------------------
        function buildSettingsMap() {
            var datasetName = document.getElementById('datasetName').value.trim();
            var username = document.getElementById('username').value.trim();
            var languageIso = document.getElementById('languageIso').value.trim();
            var altLanguage = document.getElementById('altLanguage').value.trim();
            var textData = document.getElementById('textData').value.trim();
            var audioData = document.getElementById('audioData').value.trim();
            var gordonFilter = parseInt(document.getElementById('gordonFilter').value, 10) || 0;
            var notifyOkEmails = parseEmailList(document.getElementById('notifyOk').value.trim());
            var notifyErrEmails = parseEmailList(document.getElementById('notifyErr').value.trim());

            var settings = new Map();
            settings.set('is_new', true);
            settings.set('dataset_name', datasetName);
            settings.set('username', username);
            settings.set('notify_ok', notifyOkEmails.length > 0 ? notifyOkEmails : DEFAULT_NOTIFY_OK);
            settings.set('notify_err', notifyErrEmails.length > 0 ? notifyErrEmails : DEFAULT_NOTIFY_ERR);

            if (languageIso) settings.set('language_iso', languageIso);
            if (altLanguage) settings.set('alt_language', altLanguage);

            if (textData) {
                var textDataMap = new Map();
                textDataMap.set('aws_s3', textData);
                settings.set('text_data', textDataMap);
            }
            if (audioData) {
                var audioDataMap = new Map();
                audioDataMap.set('aws_s3', audioData);
                settings.set('audio_data', audioDataMap);
            }

            var timestampsValue = document.querySelector('input[name="timestamps"]:checked').value;
            var timestamps = new Map();
            timestamps.set(timestampsValue, true);
            settings.set('timestamps', timestamps);

            var modelValue = document.querySelector('input[name="training"]:checked').value;
            var training = new Map();
            var speechToText = new Map();
            if (modelValue === 'mms_adapter') {
                training.set('mms_adapter', DEFAULT_MMS_ADAPTER);
                speechToText.set('adapter_asr', true);
            } else {
                training.set('no_training', true);
                speechToText.set('mms_asr', true);
            }
            training.set('redo_training', document.getElementById('redoTraining').checked);
            settings.set('training', training);
            settings.set('speech_to_text', speechToText);

            if (document.getElementById('compare').checked) {
                var compare = new Map();
                compare.set('html_report', true);
                compare.set('gordon_filter', gordonFilter);
                compare.set('compare_settings', DEFAULT_COMPARE_SETTINGS);
                settings.set('compare', compare);
            }

            var output = new Map();
            output.set('csv', 'yes');
            settings.set('output', output);

            return settings;
        }

        function mapToPlainValue(value) {
            if (value instanceof Map) {
                var obj = {};
                value.forEach(function (val, key) { obj[key] = mapToPlainValue(val); });
                return obj;
            }
            if (Array.isArray(value)) return value.map(mapToPlainValue);
            return value;
        }

        ---- end of commented-out old logic ---- */

        function showStatus(message, type) {
            var statusDiv = document.getElementById('status');
            statusDiv.innerHTML = message.replace(/\\n/g, '<br>');
            statusDiv.className = 'status ' + (type || 'success');
            statusDiv.style.display = 'block';
            setTimeout(function () { statusDiv.style.display = 'none'; }, 5000);
        }

        function escapeHtml(str) {
            var div = document.createElement('div');
            div.textContent = str;
            return div.innerHTML;
        }

        // ---- display validation errors one per line at the top of the
        // page. Unlike showStatus, this does not auto-hide - it stays until
        // the next validation attempt (success or failure) replaces it. ----
        function showErrors(errors) {
            var banner = document.getElementById('errorBanner');
            if (!errors || errors.length === 0) {
                banner.style.display = 'none';
                banner.innerHTML = '';
                return;
            }
            banner.innerHTML = errors.map(function (e) {
                return '<div>' + escapeHtml(e) + '</div>';
            }).join('');
            banner.style.display = 'block';
            banner.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }

        // ---- collect the raw form fields into one flat dict, no defaults,
        // no nesting, so it can be JSON.stringify'd and passed into WASM ----
        function buildFormDict() {
            var dict = {};
            dict['username'] = document.getElementById('username').value;
            dict['datasetName'] = document.getElementById('datasetName').value;
            dict['textData'] = document.getElementById('textData').value;
            dict['text_format_sfm'] = document.getElementById('text_format_sfm').checked;
            dict['text_format_usx'] = document.getElementById('text_format_usx').checked;
            dict['audioData'] = document.getElementById('audioData').value;
            dict['languageIso'] = document.getElementById('languageIso').value;
            dict['altLanguage'] = document.getElementById('altLanguage').value;
            dict['training_mms_adapter'] = document.getElementById('training_mms_adapter').checked;
            dict['redoTraining'] = document.getElementById('redoTraining').checked;
            dict['training_no_training'] = document.getElementById('training_no_training').checked;
            dict['timestamps_mms_align'] = document.getElementById('timestamps_mms_align').checked;
            dict['timestamps_mms_fa_verse'] = document.getElementById('timestamps_mms_fa_verse').checked;
            dict['compare'] = document.getElementById('compare').checked;
            dict['proofing'] = document.getElementById('proofing').checked;
            dict['gordonFilter'] = document.getElementById('gordonFilter').value;
            dict['notifyOk'] = document.getElementById('notifyOk').value;
            dict['notifyErr'] = document.getElementById('notifyErr').value;
            return dict;
        }

        function generateJSON() {
            return JSON.stringify(buildFormDict());
        }

        // ---- dropzone: walk a dropped directory down to a flat list of
        // relative file paths (root directory name included), without
        // reading any file contents or validating anything yet. ------------
        function readAllEntries(reader) {
            return new Promise(function (resolve, reject) {
                var allEntries = [];
                function readBatch() {
                    reader.readEntries(function (entries) {
                        if (entries.length === 0) {
                            resolve(allEntries);
                        } else {
                            allEntries = allEntries.concat(entries);
                            readBatch();
                        }
                    }, reject);
                }
                readBatch();
            });
        }

        function walkEntry(entry, paths) {
            if (entry.isFile) {
                paths.push(entry.fullPath.replace(/^\\//, ''));
                return Promise.resolve();
            }
            if (entry.isDirectory) {
                return readAllEntries(entry.createReader()).then(function (children) {
                    return Promise.all(children.map(function (child) {
                        return walkEntry(child, paths);
                    }));
                });
            }
            return Promise.resolve();
        }

        function setupDropzone() {
            var dropzone = document.getElementById('folderDropzone');
            var statusEl = document.getElementById('folderStatus');

            dropzone.addEventListener('dragover', function (e) {
                e.preventDefault();
                dropzone.classList.add('dragover');
            });
            dropzone.addEventListener('dragleave', function () {
                dropzone.classList.remove('dragover');
            });
            dropzone.addEventListener('drop', function (e) {
                e.preventDefault();
                dropzone.classList.remove('dragover');

                var items = e.dataTransfer.items || [];
                var entries = [];
                for (var i = 0; i < items.length; i++) {
                    var entry = items[i].webkitGetAsEntry && items[i].webkitGetAsEntry();
                    if (entry) entries.push(entry);
                }

                var paths = [];
                Promise.all(entries.map(function (entry) { return walkEntry(entry, paths); }))
                    .then(function () {
                        droppedFilePaths = paths;
                        statusEl.textContent = paths.length + ' file' + (paths.length === 1 ? '' : 's') + ' ready (validated on Save JSON)';
                        statusEl.className = 'folder-status success';
                        dropzone.classList.remove('error', 'processing');
                        dropzone.classList.add('success');
                    })
                    .catch(function (err) {
                        droppedFilePaths = [];
                        statusEl.textContent = 'Error reading dropped folder: ' + err;
                        statusEl.className = 'folder-status error';
                        dropzone.classList.remove('success', 'processing');
                        dropzone.classList.add('error');
                    });
            });
        }

        function saveJSON() {
            var jsonData = generateJSON();

            wasmReady.then(function () {
                // ValidateRequest (registered by validate.wasm's main.go,
                // which calls validate.ValidateRequestWASM) returns
                // { request: string, errors: string[] }. ValidateFiles
                // (validate.ValidateFilesWASM) takes the same dropped file
                // list - NUL-joined rather than JSON, since NUL can never
                // appear in a real filename and a plain split is simpler
                // than parsing JSON - and returns just string[] errors.
                var result = ValidateRequest(jsonData);
                var fileErrors = ValidateFiles(droppedFilePaths.join('\\0'));
                var allErrors = result.errors.concat(fileErrors);
                if (allErrors.length > 0) {
                    showErrors(allErrors);
                    return;
                }
                showErrors([]);

                var datasetName = document.getElementById('datasetName').value.trim();
                var filename = (datasetName ? datasetName : 'request') + '.json';

                var blob = new Blob([jsonData], { type: 'application/json;charset=utf-8' });
                var url = URL.createObjectURL(blob);
                var a = document.createElement('a');
                a.href = url;
                a.download = filename;
                a.style.display = 'none';
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                setTimeout(function () { URL.revokeObjectURL(url); }, 1000);

                showStatus('Saved: ' + filename);
            }).catch(function (err) {
                showErrors(['Validation module failed to load: ' + err]);
            });
        }

        function updateRedoTrainingState() {
            var mmsAdapterSelected = document.getElementById('training_mms_adapter').checked;
            var redoTraining = document.getElementById('redoTraining');
            redoTraining.disabled = !mmsAdapterSelected;
            if (!mmsAdapterSelected) redoTraining.checked = false;
        }

        function clearForm() {
            var username = document.getElementById('username').value;

            document.getElementById('requestForm').reset();
            document.getElementById('gordonFilter').value = '0';
            document.getElementById('timestamps_mms_align').checked = true;
            document.getElementById('training_mms_adapter').checked = true;
            document.getElementById('compare').checked = true;
            document.getElementById('text_format_sfm').checked = true;
            document.getElementById('notifyOk').value = 'ewallace@fcbhmail.org, gfiddes@fcbhmail.org, edomschot@fcbhmail.org';
            document.getElementById('notifyErr').value = 'gary@shortsands.com, ewallace@fcbhmail.org';

            document.getElementById('username').value = username;

            var folderDropzone = document.getElementById('folderDropzone');
            folderDropzone.innerHTML =
                '<div>\\uD83D\\uDCC1 Drag a media folder or YAML file here, or press to select</div>' +
                '<div class="folder-status" id="folderStatus">No folder or file selected</div>' +
                '<div class="folder-progress" id="folderProgress" style="display: none;"></div>';
            folderDropzone.classList.remove('success', 'error', 'processing');
            droppedFilePaths = [];

            updateRedoTrainingState();
            showErrors([]);
        }

        window.clearForm = clearForm;
        window.saveJSON = saveJSON;

        document.addEventListener('DOMContentLoaded', function () {
            updateRedoTrainingState();
            setupDropzone();

            /* ---- old required/optional field validation wiring: commented
            out along with the validation functions above.

            updateRequiredFieldStyling();

            var requiredFields = ['datasetName', 'username', 'languageIso', 'textData', 'audioData'];
            var optionalFields = ['notifyOk', 'notifyErr'];
            var allValidationFields = requiredFields.concat(optionalFields);
            var validationTimers = {};

            allValidationFields.forEach(function (fieldId) {
                var element = document.getElementById(fieldId);

                element.addEventListener('input', function () {
                    element.classList.remove('required-error');
                    if (validationTimers[fieldId]) clearTimeout(validationTimers[fieldId]);
                    validationTimers[fieldId] = setTimeout(function () {
                        var value = element.value.trim();
                        if (!value) return;
                        var isValid = true;
                        if (fieldId === 'languageIso' && !validateLanguageISO(value)) isValid = false;
                        else if ((fieldId === 'textData' || fieldId === 'audioData') && !validateS3URL(value)) isValid = false;
                        else if ((fieldId === 'notifyOk' || fieldId === 'notifyErr') && !validateEmailList(value)) isValid = false;
                        if (!isValid) element.classList.add('required-error');
                    }, 3000);
                    if (requiredFields.indexOf(fieldId) !== -1) updateRequiredFieldStyling();
                });

                element.addEventListener('blur', function () {
                    if (validationTimers[fieldId]) clearTimeout(validationTimers[fieldId]);
                    if (requiredFields.indexOf(fieldId) !== -1) updateRequiredFieldStyling();
                });
            });

            ---- end of commented-out validation wiring ---- */

            document.querySelectorAll('input[name="text_format"]').forEach(function (radio) {
                radio.addEventListener('change', function () {
                    var textData = document.getElementById('textData');
                    if (textData.value) {
                        textData.value = textData.value.replace(/\\*\\.(usx|sfm)/i, '*.' + this.value);
                    }
                });
            });

            document.querySelectorAll('input[name="training"]').forEach(function (radio) {
                radio.addEventListener('change', updateRedoTrainingState);
            });
        });
    })();
    </script>
</body>

</html>`;
