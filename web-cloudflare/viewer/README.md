# Arti R2 Viewer

A read-only Cloudflare Worker at `/viewer` for browsing the three arti R2
buckets (`arti-input`, `arti-models`, `arti-output`) without needing S3
tooling. See `viewer.spec` for the full behavior spec this implements.

## Files

- `viewer.toml` - wrangler config; binds the three buckets natively (`env.ARTI_INPUT` etc.), no S3 credentials needed.
- `viewer.js` - routing: serves the page, the JSON list endpoints, and `/viewer/file` (download / show / play streaming, including HTTP Range support for audio).
- `r2_list.js` - paginates `bucket.list()` and aggregates keys into the distinct-tuple rows each tab needs.
- `output_details.js` - builds the fixed-row Details table for one arti-output run, per the confirmed layout in `courier/courier.go` (see comments at the top of that file: run_num is zero-padded to 5 digits, "Output1..Outputn"/multiple databases are numbered by this Worker rather than being real subfolders, and Runtime/Duration are read from the object *key* itself, not a file body).
- `viewer_page.js` - the page itself: plain HTML/CSS/JS, no build step, no external scripts.

## Deploy

```
wrangler deploy -c viewer.toml
```

Then add a route in `viewer.toml`, e.g.:

```toml
routes = [
  { pattern = "internal.arti.example/viewer*", zone_name = "arti.example" }
]
```

## Known gaps / things to verify against the real buckets

- **No auth.** The spec didn't call for it, but these buckets hold internal
  pipeline input/output. Consider putting this behind Cloudflare Access, the
  same way `web-cloudflare/upload` does, before exposing it beyond
  localhost/VPN.
- **Timestamps.** Per your call, only R2's single `uploaded` timestamp is
  used everywhere - there's no created-vs-modified distinction.
