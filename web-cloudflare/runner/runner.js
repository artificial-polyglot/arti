// runner.js
// Cloudflare Worker serving the job-request form: a browser page that
// collects a media/job request and lets the user download it as JSON.
// The form fields are lifted as-is from arti/controller/web/arti.html; the
// old AWS-S3-upload/YAML client (arti-main.js, arti-upload.js,
// arti-validation.js) is not used here - all form-reading logic below is
// new and produces JSON instead of YAML.
//
// Routes:
//   GET  /         - the page itself
//   GET  /arti.png - the logo, served from ./public via the ASSETS binding

import { PAGE_HTML } from "./runner_page.js";

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (url.pathname === "/") {
      return new Response(PAGE_HTML, { headers: { "content-type": "text/html; charset=utf-8" } });
    }
    return env.ASSETS.fetch(request);
  },
};
