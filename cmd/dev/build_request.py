import json
import os
import sys


def main():
    if len(sys.argv) != 2:
        print("usage: build_request.py <path-to-request.yaml>", file=sys.stderr)
        sys.exit(1)

    yaml_path = sys.argv[1]

    # Read the YAML file's contents as a string
    try:
        with open(yaml_path, "r") as f:
            yaml_text = f.read()
    except OSError as e:
        print(f"cannot read {yaml_path}: {e}", file=sys.stderr)
        sys.exit(1)

    # Build the structure the handler expects
    request = {
        "input": {
            "request_yaml": yaml_text
        }
    }

    # Write it as a JSON file alongside the input, with a .json extension
    out_path = os.path.splitext(yaml_path)[0] + ".json"
    with open(out_path, "w") as f:
        json.dump(request, f, indent=2)

    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()