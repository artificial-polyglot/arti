Let me verify the exact aws4fetch signing API against R2's expectations before handing you code, so you don't re-fight the signature mismatch.Verified. `X-Amz-SignedHeaders = host` (so a browser-added Content-Type stays unsigned and won't trip the signature), `X-Amz-Expires` is picked up from the query param, and the key path with slashes survives intact. That's the exact behavior we want. Writing the two files now.Both files are above. The signing call in the Worker is the one I test-verified, so it should work against R2 on the first try. Here's the wiring you still need to do outside the code:

**Worker secrets/vars** (in `wrangler.toml` under `[vars]` for the non-secret ones, and `wrangler secret put` for the two keys):
```
R2_ACCOUNT_ID, R2_BUCKET, ALLOWED_ORIGIN, UPLOAD_TOKEN   # vars
R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY                    # secrets
```
The two keys come from an R2 API token (R2 → Manage API Tokens), with Object Read & Write on the target bucket. Note there's no R2 binding in the `wrangler.toml` here on purpose — presigning uses the S3 endpoint + credentials, not a bucket binding.

**R2 bucket CORS** — this is the piece that's separate from the Worker's CORS and the usual reason a valid URL still fails. Set it on the bucket (R2 → your bucket → Settings → CORS):
```json
[
  {
    "AllowedOrigins": ["https://your-app.example.com"],
    "AllowedMethods": ["PUT"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```
Use your real app origin (where `upload-client.html` is served from). `["*"]` for `AllowedOrigins` works during local testing but tighten it before production.

**How the two connect:** open the client, point "Presign endpoint" at your deployed Worker URL, set the prefix (e.g. `langXYZ/NT`) and token, choose the folder. The client keeps each file's path under the chosen folder (`043-John/043-John-003.wav`) and appends it to the prefix to form the R2 key.

A few design choices worth knowing so you can tune them:

- It mints all URLs in one batch call, as you asked, with a 3600 s expiry. For ~260 chapter files at a few MB each that's comfortable even on a slow uplink. If you ever batch something huge or expect very slow connections, the alternative is minting just-in-time inside the upload pool (mint right before each PUT) so expiry never matters — easy to switch since minting and uploading are already decoupled.
- Concurrency is capped at 6 parallel PUTs (`CONCURRENCY`), with 3 retries and linear backoff per file. Failures are marked in the list and a re-run only re-uploads what didn't succeed... well, currently a re-run re-uploads everything selected — if you want "retry failures only," that's a small filter on the states map I can add.
- Progress is per-file count, not per-byte. `fetch` can't report upload progress; if you want a real byte-level bar, that one PUT needs to move to `XMLHttpRequest` with `upload.onprogress`. Say the word and I'll swap that in.

One thing I did not build in, since it depends on your pipeline: after a successful upload you'll probably want the client (or a second Worker route) to notify Arti that new audio landed — e.g. drop a row in the QA SQLite manifest or kick the RunPod job. Right now it just lands the objects in R2.