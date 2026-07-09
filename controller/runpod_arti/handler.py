import subprocess
import runpod

DEFAULT_TIMEOUT=60

def handler(job):
    job_input = job["input"]
    request_text = job_input["request_yaml"]
    timeout = job_input.get("timeout_minutes", DEFAULT_TIMEOUT)
    timeout *= 60 # convert to seconds
    try:
        result = subprocess.run(
            ["/app/runpod_arti", request_text],
            text=True, timeout=timeout,
        )
    except subprocess.TimeoutExpired as e:
        return {"error": f"process timed out after {e.timeout} seconds"}

    if result.returncode != 0:
        return {"error": result.stderr.strip(), "stdout": result.stdout.strip()}
    return {"status": "complete", "output": result.stdout.strip()}

runpod.serverless.start({"handler": handler})