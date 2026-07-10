import json
import os
import sys
import time
import requests

def main():
    if len(sys.argv) < 2:
        print("usage: build_request.py <path-to-request.yaml>", file=sys.stderr)
        sys.exit(1)
    yaml_path = sys.argv[1]
    if len(sys.argv) == 3:
        run_local = False
    else:
        run_local = True

    # Read the YAML file's contents as a string
    try:
        with open(yaml_path, "r") as f:
            yaml_text = f.read()
    except OSError as e:
        print(f"cannot read {yaml_path}: {e}", file=sys.stderr)
        sys.exit(1)

    # Build the structure the handler expects
    payload = {
        "input": {
            "request_yaml": yaml_text,
            "timeout_minutes": 120
        }
    }

    if run_local:
        out_path = os.path.join(os.path.dirname(yaml_path), "test_input.json")
        with open(out_path, "w") as f:
            json.dump(payload, f, indent=2)
        print(f"wrote {out_path}")

    else:
        endpoint = "fzfyyzux8a5dhi"
        api_key = os.environ["RUNNING_PHESANT"]

        url = f"https://api.runpod.ai/v2/{endpoint}/run"
        headers = {
            "Content-Type": "application/json",
            "Authorization": api_key,
        }
        response = requests.post(url, headers=headers, json=payload)
        response.raise_for_status()
        print(response.json())
        job = response.json()
        job_id = job["id"]
        print(f"submitted job {job_id}")
        requests.post("https://ntfy.sh/arti3", data=f"Job {job_id} submitted: {yaml_path}")

        # Poll for completion
        status_url = f"https://api.runpod.ai/v2/{endpoint}/status/{job_id}"
        while True:
            time.sleep(30)
            status_response = requests.get(status_url, headers=headers)
            status_response.raise_for_status()
            status = status_response.json()
            print(status["status"])
            if status["status"] in ("COMPLETED", "FAILED"):
                break

        print(status)
        requests.post("https://ntfy.sh/arti3", data=f"Job {job_id} finished: {status['status']}")

if __name__ == "__main__":
    main()