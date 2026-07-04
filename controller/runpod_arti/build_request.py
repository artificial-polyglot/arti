import json
import os
import sys
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
            "request_yaml": yaml_text
        }
    }

    if run_local:
        # Write it as a JSON file alongside the input, with a .json extension
        out_path = os.path.splitext(yaml_path)[0] + ".json"
        with open(out_path, "w") as f:
            json.dump(payload, f, indent=2)
        print(f"wrote {out_path}")

    else:
        endpoint = "c9ct1wa5n9wxyq"  # RunPod endpoint ID
        api_key = os.environ["RUNPOD_API_KEY"]

        url = f"https://api.runpod.ai/v2/{endpoint}/runsync"
        headers = {
            "Content-Type": "application/json",
            "Authorization": api_key,
        }

        #payload = {"input": {"request_yaml": "..."}}  # your constructed JSON goes here

        response = requests.post(url, headers=headers, json=payload)
        response.raise_for_status()
        print(response.json())

if __name__ == "__main__":
    main()