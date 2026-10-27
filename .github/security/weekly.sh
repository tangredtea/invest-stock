#!/usr/bin/env bash
set -euo pipefail
umask 077
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
project=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}
[[ "$project" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]
python3 - "$script_dir/project.json" "$project" <<'PY'
import json,sys
assert json.load(open(sys.argv[1]))['project']==sys.argv[2], 'Project configuration mismatch'
PY
tool_root="${RUNNER_TOOL_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}}/security-fleet"
root="$tool_root/${project//\//_}"
install -d -m 700 "$root/reports" "$tool_root/cache"
exec 9>"$root/scan.lock"
flock -w 1200 9
# One owned DB per runner, locked across readers/updaters; not 21 duplicate databases.
exec 8>"$tool_root/cache.lock"
flock -w 1200 8
trivy=$(bash "$script_dir/ensure-trivy.sh")
output="$root/reports/${GITHUB_RUN_ID:?}-${GITHUB_RUN_ATTEMPT:-1}-${GITHUB_JOB:-audit}-$(date -u +%Y%m%dT%H%M%S)-$$"
status=0
python3 "$script_dir/scan.py" source "$GITHUB_WORKSPACE" --project "$project" \
  --output "$output" --cache "$tool_root/cache" --trivy "$trivy" || status=$?
notify_status=0
python3 - "$output" "$GITHUB_STEP_SUMMARY" "$root" "${SECURITY_FLEET_MODE:-release}" "$script_dir/project.json" <<'PY' || notify_status=$?
import base64, datetime as dt, gzip, json, os, sys
from pathlib import Path
p=Path(sys.argv[1]); result=json.loads((p/('summary.json' if (p/'summary.json').exists() else 'failure.json')).read_text())
root=Path(sys.argv[3]); mode=sys.argv[4]
config=json.loads(Path(sys.argv[5]).read_text())
manifest=config.get('exceptionsManifest')
if manifest:
 approval=json.loads((Path(os.environ['GITHUB_WORKSPACE'])/manifest).read_text())['approval']
 expiry=dt.datetime.fromisoformat(approval['expiresAt'].replace('Z','+00:00'))
 remaining=expiry-dt.datetime.now(dt.timezone.utc)
 result['exceptionWindow']={'expiresAt':approval['expiresAt'],
   'state':'expired' if remaining.total_seconds()<=0 else 'expiring' if remaining<=dt.timedelta(days=7) else 'active'}
previous={}
latest=root/'weekly-latest.json'
if latest.exists():previous=json.loads(latest.read_text())
def important(row):
 return row['severity'] in ('HIGH','CRITICAL')
def key(row):
 return tuple(row.get(k,'') for k in ('id','package','version','purl','severity','fixedVersion'))
old={key(row) for row in previous.get('findings',[]) if important(row)}
new={key(row) for row in result.get('findings',[]) if important(row)}
window_changed=result.get('exceptionWindow')!=previous.get('exceptionWindow')
notify=bool(new-old) or window_changed or result['status']=='incomplete'
result['notificationRequired']=notify
if mode=='weekly':
 temp=root/f"weekly-latest-{os.environ['GITHUB_RUN_ID']}-{os.environ.get('GITHUB_RUN_ATTEMPT','1')}.json"
 with open(temp,'x') as f:json.dump(result,f)
 os.chmod(temp,0o600);os.replace(temp,latest)
payload=base64.b64encode(gzip.compress(json.dumps(result,separators=(',',':')).encode())).decode()
print('SECURITY_FLEET_REPORT_V1='+payload)
with open(sys.argv[2],'a') as f:
 f.write(f"## Weekly security scan\n\nStatus: **{result['status']}**\n\n")
 f.write(f"Project: `{result['project']}`\n\n")
 f.write(f"Packages: {result.get('packageCount','unknown')}; findings: `{json.dumps(result.get('counts',{}))}`\n\n")
 f.write('Source dependency audit only. Running images are scanned separately by the fleet host timer. No dependencies installed, application started, deployment or automatic risk acceptance.\n')
 if mode=='weekly':f.write(f"\nMeaningful change requiring notification: **{notify}**. A completed audit with unchanged findings does not mean the findings are fixed.\n")
if mode=='weekly' and notify:sys.exit(3)
PY
if [[ "$notify_status" != 0 && "$notify_status" != 3 ]]; then
  echo 'Safe security summary could not be completed.' >&2
  exit 2
fi
# Release gates always block HIGH/CRITICAL; weekly jobs alert only on meaningful changes.
if [[ "${SECURITY_FLEET_MODE:-release}" == weekly && "$status" != 2 ]]; then
  # The centralized follow-up owns change-only alerts across different runner nodes.
  # A completed weekly audit retains status=findings instead of turning accepted history green.
  exit 0
fi
exit "$status"
