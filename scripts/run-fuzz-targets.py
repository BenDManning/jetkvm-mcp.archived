#!/usr/bin/env python3
"""Run the checked Go fuzz-target inventory without shell interpolation."""

import argparse
import json
import os
import pathlib
import re
import subprocess
import tempfile


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fuzztime", required=True)
    args = parser.parse_args()
    if not re.fullmatch(r"(?:[1-9][0-9]*x|[1-9][0-9]*(?:ms|s|m))", args.fuzztime):
        parser.error("--fuzztime must be a positive iteration count or duration")

    root = pathlib.Path(__file__).resolve().parent.parent
    entries = json.loads((root / "testdata" / "fuzz-targets.json").read_text(encoding="utf-8"))
    with tempfile.TemporaryDirectory(prefix="jetkvm-mcp-fuzz-cache-") as cache:
        environment = os.environ.copy()
        environment["GOCACHE"] = cache
        for entry in entries:
            if set(entry) != {"package", "target"}:
                raise SystemExit("invalid fuzz-target manifest fields")
            package = entry["package"]
            target = entry["target"]
            if not re.fullmatch(r"\./internal/[a-z0-9_/]+", package) or not re.fullmatch(r"Fuzz[A-Za-z0-9_]+", target):
                raise SystemExit("invalid fuzz-target manifest value")
            print(f"fuzz smoke: {package} {target}", flush=True)
            subprocess.run(
                ["go", "test", package, "-run", "^$", "-fuzz", f"^{target}$", "-fuzztime", args.fuzztime, "-parallel", "1"],
                cwd=root,
                env=environment,
                check=True,
            )


if __name__ == "__main__":
    main()
