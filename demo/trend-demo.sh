#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

DB="/tmp/sift_demo_trend.db"
rm -f "$DB"

pause() { sleep "${1:-1.5}"; }

type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.04
    done
    echo
}

ingest_at() {
    local file="$1" source="$2" offset_days="$3"
    python3 -c "
import base64, json, os, subprocess, sqlite3
from datetime import datetime, timedelta, timezone

with open('$file', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'$source'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

# Backdate the report in the DB
report_id = None
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            report_id = json.loads(obj['result']['content'][0]['text'])['report_id']
            break
    except: pass

if report_id:
    backdated = (datetime.now(timezone.utc) - timedelta(days=$offset_days)).isoformat()
    conn = sqlite3.connect('$DB')
    # Patch both the column and the embedded Timestamp in raw_json
    row = conn.execute('SELECT raw_json FROM test_reports WHERE id = ?', (report_id,)).fetchone()
    if row:
        raw = json.loads(row[0])
        raw['Timestamp'] = backdated
        patched = json.dumps(raw)
        conn.execute('UPDATE test_reports SET timestamp = ?, created_at = ?, raw_json = ? WHERE id = ?', (backdated, backdated, patched, report_id))
    conn.execute('UPDATE test_failures SET timestamp = ?, created_at = ? WHERE report_id = ?', (backdated, backdated, report_id))
    conn.commit()
    conn.close()
    print(f'  Ingested run ($source, -{$offset_days}d): report_id={report_id[:8]}...')
"
}

clear
echo "=== Sift: Severity Trend ==="
echo "Tracking how failure severity changes across CI runs over time"
echo ""
pause 2

echo "# Simulating a week of CI runs (7 reports ingested)"
pause 1
echo ""

ingest_at testdata/cbioportal-report.xml  "api-ci" 6
ingest_at testdata/flaky-run-1.xml        "api-ci" 5
ingest_at testdata/flaky-run-2.xml        "api-ci" 4
ingest_at testdata/pytest-report.xml      "api-ci" 3
ingest_at testdata/flaky-run-1.xml        "api-ci" 2
ingest_at testdata/pytest-report.xml      "api-ci" 1
ingest_at testdata/flaky-run-2.xml        "api-ci" 0

pause 2

echo ""
echo "# Query severity trend over the last 7 days"
type_cmd "mcp__sift__get_severity_trend"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_severity_trend','arguments':{'time_range':'7d','bucket':'day'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_trend.db'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Time range: {r[\"time_range\"]}  |  Bucket: {r[\"bucket\"]}')
            print()
            for date in sorted(r.get('trend', {}).keys()):
                severities = r['trend'][date]
                bar = ''
                for sev in ['critical','high','medium','low']:
                    count = severities.get(sev, 0)
                    if count:
                        bar += f'{sev}:{count}  '
                print(f'  {date}  {bar}')
    except: pass
"
pause 3

echo ""
echo "# Spikes visible at a glance."
echo "# Pipe this into your incident dashboard or let Claude triage it."
pause 3
