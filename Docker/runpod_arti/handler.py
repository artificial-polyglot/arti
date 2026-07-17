import subprocess
import sys
import threading
import runpod

DEFAULT_TIMEOUT = 60

def handler(job):
    job_input = job["input"]
    request_text = job_input["request_yaml"]
    timeout = job_input.get("timeout_minutes", DEFAULT_TIMEOUT)
    timeout *= 60  # convert to seconds

    #appToRun = "/app/runpod_arti"
    appToRun = "/app/qa_align"
    try:
        result = subprocess.run(
            [appToRun, request_text],
            stdout=subprocess.PIPE,
            stderr=None,
            text=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        return {"error": f"process timed out after {timeout} seconds"}

    stdout = result.stdout.strip()
    if result.returncode != 0:
        return {
            "error": "process failed",
            "returncode": result.returncode,
            "stdout": stdout,
        }
    return {"status": "complete", "output": stdout}

runpod.serverless.start({"handler": handler})
