# runner

Cloudflare Worker serving the job-request form at `https://run.arti2.workers.dev`.

The form markup/styling is carried over as-is from `arti/controller/web/arti.html`,
minus the AWS-credentials dropzone and the Upload/YAML-load machinery that only
made sense with the old S3-upload client (`arti-main.js`, `arti-upload.js`,
`arti-validation.js` — none of that is used here). The folder dropzone markup
is kept in place for a later pass but isn't wired to any JS yet.

`runner_page.js` contains the page's own script: it reads the form fields into
a `Map` (with nested `Map`s for grouped settings like `training`, `timestamps`,
`compare`), flattens that into plain objects, and downloads it as JSON via the
**Save JSON** button. No network calls yet — this is just the form -> JSON step.

## Files

- `runner.toml` — Worker config; serves `public/arti.png` via the `ASSETS` binding.
- `runner.js` — routes `/` to the page, everything else to `ASSETS`.
- `runner_page.js` — the page itself (HTML/CSS/JS as one exported string).
- `public/arti.png` — the logo, copied from `arti/controller/web/arti.png`.

## Local dev

```
wrangler dev -c runner.toml
```

## Deploy

```
wrangler deploy -c runner.toml
```
