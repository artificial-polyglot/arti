# Arti R2 Viewer

A read-only Cloudflare Worker at `https://view.arti2.workers.dev` for browsing
the three arti R2 buckets (`arti-input`, `arti-models`, `arti-output`)
without needing S3 tooling. See `viewer.spec` for the full behavior spec this
implements.

## Files

- `viewer.toml` - wrangler config; binds the three buckets natively (`env.ARTI_INPUT` etc.), no S3 credentials needed. Also names the `AUDIO_SIGN_SECRET` secret (provisioned separately via `wrangler secret put`, not stored in the file) used to sign report-download audio links.
- `viewer.js` - routing: serves the page, the JSON list endpoints, `/file` (download / show / play streaming, including HTTP Range support for audio, and optional `exp`+`sig` signature verification), and `/api/sign-audio-urls` (mints signed `/file` links for a report's `audio_file_urls.json` manifest, so a downloaded HTML report - `match/diff/html_writer.go` or `cmd/output/proofing_rpt/html_writer.go` - keeps its audio playable wherever it ends up, for 7 days).
- `r2_list.js` - paginates `bucket.list()` and aggregates keys into the distinct-tuple rows each tab needs.
- `output_details.js` - builds the fixed-row Details table for one arti-output run, per the confirmed layout in `courier/courier.go` (see comments at the top of that file: run_num is zero-padded to 5 digits, "Output1..Outputn"/multiple databases are numbered by this Worker rather than being real subfolders, and Runtime/Duration are read from the object *key* itself, not a file body).
- `viewer_page.js` - the page itself: plain HTML/CSS/JS, no build step, no external scripts.

## Deploy

```
wrangler deploy -c viewer.toml
```

Deploys to the account's `workers.dev` subdomain as `view.arti2.workers.dev`
(the worker is named `view` in `viewer.toml`, and the account subdomain is
`arti2`). To serve on a custom zone instead, add a route in `viewer.toml`,
e.g.:

```toml
routes = [
  { pattern = "internal.arti.example/*", zone_name = "arti.example" }
]
```

## Known gaps / things to verify against the real buckets

- **No auth.** The spec didn't call for it, but these buckets hold internal
  pipeline input/output. Consider putting this behind Cloudflare Access, the
  same way `web-cloudflare/upload` does, before exposing it beyond
  localhost/VPN.
- **Timestamps.** Per your call, only R2's single `uploaded` timestamp is
  used everywhere - there's no created-vs-modified distinction.
