import subprocess
import sys
import threading
import runpod

DEFAULT_TIMEOUT = 60


def _pump(pipe, sink, echo=False):
    for line in iter(pipe.readline, ""):
        if echo:
            print(line, end="", file=sys.stderr, flush=True)
        sink.append(line)
    pipe.close()


def handler(job):
    job_input = job["input"]
    request_text = job_input["request_yaml"]
    timeout = job_input.get("timeout_minutes", DEFAULT_TIMEOUT)
    timeout *= 60  # convert to seconds

    proc = subprocess.Popen(
        ["/app/runpod_arti", request_text],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,  # line buffered
    )

    # Read stdout and stderr concurrently in their own threads. This avoids
    # a deadlock where the child fills an OS pipe buffer while nobody is
    # draining it, which would otherwise hang the worker indefinitely.
    stdout_lines, stderr_lines = [], []
    out_thread = threading.Thread(target=_pump, args=(proc.stdout, stdout_lines), daemon=True)
    err_thread = threading.Thread(target=_pump, args=(proc.stderr, stderr_lines, True), daemon=True)
    out_thread.start()
    err_thread.start()

    try:
        returncode = proc.wait(timeout=timeout)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait()
        out_thread.join(timeout=5)
        err_thread.join(timeout=5)
        return {"error": f"process timed out after {timeout} seconds"}

    # Process has exited; the pipes will hit EOF once drained, so this
    # blocks only long enough to flush whatever's left in the OS buffers.
    out_thread.join(timeout=5)
    err_thread.join(timeout=5)

    stdout = "".join(stdout_lines).strip()
    if returncode != 0:
        return {
            "error": "process failed",
            "returncode": returncode,
            "stdout": stdout,
            "stderr": "".join(stderr_lines).strip(),
        }
    return {"status": "complete", "output": stdout}


runpod.serverless.start({"handler": handler})
