#!/usr/bin/env python3
"""Read-only, dependency-free fleet scanner. Exit 0=clear/N/A, 1=findings, 2=incomplete."""
from __future__ import annotations

import argparse
import collections
import datetime as dt
import json
import os
from pathlib import Path
import re
import subprocess

VERSION = "1.0.1"
TRIVY_VERSION = "0.70.0"
SKIP_DIRS = [".git", "node_modules", ".venv", "venv", "vendor", ".wasp", "dist", "build", ".next"]
LOCKS = {"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb", "go.mod", "go.sum", "poetry.lock", "uv.lock", "Pipfile.lock", "Cargo.lock", "composer.lock", "Gemfile.lock", "packages.lock.json"}


class IncompleteCoverage(ValueError):
    pass


def run(args, **kwargs):
    return subprocess.run(args, check=True, text=True, capture_output=True, **kwargs).stdout.strip()


def save(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    with open(path, "x", encoding="utf-8") as stream:
        json.dump(value, stream, ensure_ascii=False, indent=2)
        stream.write("\n")
    path.chmod(0o600)


def summarize(report, *, project, target, kind, manifests=None):
    if not isinstance(report, dict) or not isinstance(report.get("Results"), list):
        raise ValueError("Missing scanner Results: coverage is unknown")
    rows, packages, types = [], 0, set()
    for result in report["Results"]:
        if not isinstance(result, dict):
            raise ValueError("Malformed scanner result")
        packages += len(result.get("Packages") or [])
        types.add(result.get("Type", "unknown"))
        for row in result.get("Vulnerabilities") or []:
            if not all(row.get(k) for k in ("VulnerabilityID", "PkgName", "InstalledVersion", "Severity")):
                raise ValueError("Malformed vulnerability row")
            # Deliberate allowlist: never copy ImageConfig, descriptions or environment.
            rows.append({"id": row["VulnerabilityID"], "package": row["PkgName"],
                         "version": row["InstalledVersion"], "severity": row["Severity"],
                         "fixedVersion": row.get("FixedVersion", ""),
                         "purl": (row.get("PkgIdentifier") or {}).get("PURL", ""),
                         "type": result.get("Type", "unknown")})
    rows = sorted({json.dumps(row, sort_keys=True): row for row in rows}.values(),
                  key=lambda row: (row["id"], row["package"], row["version"], row["purl"]))
    counts = dict(collections.Counter(row["severity"] for row in rows))
    status = "findings" if counts.get("HIGH", 0) + counts.get("CRITICAL", 0) else "clear"
    if kind == "source" and not manifests:
        status = "not-applicable"
    elif not packages:
        raise IncompleteCoverage("Expected dependencies/image packages but scanner detected no installed packages")
    return {"schemaVersion": 1, "toolkitVersion": VERSION, "project": project,
            "kind": kind, "target": target, "status": status,
            "scannedAt": dt.datetime.now(dt.timezone.utc).isoformat(),
            "scannerVersion": TRIVY_VERSION, "packageCount": packages,
            "detectedTypes": sorted(types), "manifests": manifests or [],
            "counts": counts, "findings": rows,
            "riskExceptionsApplied": False}


def changes(previous, current):
    def identity(row):
        return tuple(row.get(k, "") for k in ("id", "package", "version", "purl", "severity", "fixedVersion"))
    old = {identity(row) for row in (previous or {}).get("findings", [])}
    new = {identity(row) for row in current.get("findings", [])}
    return {"addedOrChanged": [list(row) for row in sorted(new - old)],
            "resolvedOrChanged": [list(row) for row in sorted(old - new)]}


def source_manifests(root):
    files = run(["git", "-C", str(root), "ls-files", "-z"]).split("\0")
    return [f for f in files if f and not any(part in SKIP_DIRS for part in Path(f).parts)
            and (Path(f).name in LOCKS or (Path(f).name.startswith("requirements") and f.endswith(".txt")))]


def scanner_env(cache):
    # Do not inherit TRIVY_IGNORE_UNFIXED, severity filters or scanner overrides.
    return {**{k: v for k, v in os.environ.items() if not k.startswith("TRIVY_")},
            "TRIVY_CACHE_DIR": str(cache)}


def scan(kind, target, project, output, cache, trivy="trivy"):
    output = Path(output).resolve()
    output.mkdir(parents=True, exist_ok=False, mode=0o700)
    os.chmod(output, 0o700)
    env = scanner_env(cache)
    actual = run([trivy, "--config", "/dev/null", "--version"], env=env)
    if f"Version: {TRIVY_VERSION}" not in actual.splitlines():
        raise ValueError("Scanner version does not match the fleet pin")
    manifests = None
    if kind == "source":
        root = Path(target).resolve()
        if run(["git", "-C", str(root), "status", "--porcelain", "--untracked-files=no"]):
            raise ValueError("Source scan requires a clean tracked checkout for accurate SHA attribution")
        manifests = source_manifests(root)
        target_identity = run(["git", "-C", str(root), "rev-parse", "HEAD"])
        command = [trivy, "fs", str(root)]
    else:
        target_identity = run(["docker", "image", "inspect", "--format", "{{.Id}}", target])
        if not re.fullmatch(r"sha256:[0-9a-f]{64}", target_identity):
            raise ValueError("Cannot resolve an immutable local image identity")
        command = [trivy, "image", "--image-src", "docker", target_identity]
    raw = output / "raw.json"
    command[2:2] = ["--config", "/dev/null", "--scanners", "vuln", "--list-all-pkgs",
                    "--severity", "UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL", "--ignore-unfixed=false",
                    "--ignorefile", "/dev/null", "--exit-code", "0", "--timeout", "20m",
                    "--db-repository", "ghcr.io/aquasecurity/trivy-db:2",
                    "--format", "json", "--output", str(raw)]
    if kind == "source":
        for directory in SKIP_DIRS:
            command[2:2] = ["--skip-dirs", directory]
        command[2:2] = ["--skip-files", "**/.env*", "--skip-files", ".env*"]
    run(command, env=env)
    raw.chmod(0o600)
    report = json.loads(raw.read_text())
    summary = summarize(report, project=project, target=target_identity, kind=kind, manifests=manifests)
    if kind == "image" and report.get("Metadata", {}).get("ImageID") != target_identity:
        raise ValueError("Scan report does not match the inspected image")
    db_path = Path(cache) / "db/metadata.json"
    db = json.loads(db_path.read_text())
    updated = dt.datetime.fromisoformat(db["UpdatedAt"].replace("Z", "+00:00"))
    if not dt.timedelta(minutes=-5) <= dt.datetime.now(dt.timezone.utc) - updated <= dt.timedelta(hours=48):
        raise ValueError("Vulnerability database is stale or has an invalid timestamp")
    summary["databaseUpdatedAt"] = db["UpdatedAt"]
    save(output / "summary.json", summary)
    return summary


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("kind", choices=["source", "image"])
    parser.add_argument("target")
    parser.add_argument("--project", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--cache", required=True)
    parser.add_argument("--trivy", default="trivy")
    args = parser.parse_args(argv)
    os.umask(0o077)
    try:
        result = scan(args.kind, args.target, args.project, args.output, args.cache, args.trivy)
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        # Do not echo command output: scanner or registry messages may contain credentials.
        result = {"schemaVersion": 1, "project": args.project, "kind": args.kind,
                  "status": "incomplete", "errorType": type(error).__name__,
                  "scannedAt": dt.datetime.now(dt.timezone.utc).isoformat()}
        if isinstance(error, IncompleteCoverage):
            result["reason"] = "dependency-inventory-empty"
        failure = Path(args.output) / "failure.json"
        if not failure.exists():
            save(failure, result)
        print(json.dumps(result, ensure_ascii=False))
        return 2
    print(json.dumps({k: v for k, v in result.items() if k not in {"findings", "manifests"}}, ensure_ascii=False))
    return int(result["status"] == "findings")


if __name__ == "__main__":
    raise SystemExit(main())
