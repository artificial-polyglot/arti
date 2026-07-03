import os
import tempfile
import subprocess
import runpod

def handler(job):
    job_input = job["input"]
    request_text = job_input["request_yaml"]  # the raw contents of your request.yaml, as a string

    result = subprocess.run(
        ["/app/hello_arti", request_text],   # <-- your binary name
        capture_output=True, text=True, timeout=300,
    )

    if result.returncode != 0:
        return {"error": result.stderr.strip(), "stdout": result.stdout.strip()}
    return {"status": "complete", "output": result.stdout.strip()}

runpod.serverless.start({"handler": handler})
