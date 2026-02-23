#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

DB="/tmp/sift_demo_flaky.db"
rm -f "$DB"

type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.04
    done
    echo
}

pause() { sleep "${1:-1.5}"; }

run_tool() {
    local tool="$1" args="$2"
    python3 -c "
import base64, json, os, subprocess

def call(tool, args, db):
    init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
    call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name': tool,'arguments': args}})
    env = os.environ.copy()
    env['SIFT_DB_PATH'] = db
    proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
    for line in proc.stdout.strip().split('\n'):
        try:
            obj = json.loads(line)
            if obj.get('id') == 2:
                return json.loads(obj['result']['content'][0]['text'])
        except: pass
    return None

tool = '$tool'
args = $args
db = '$DB'
result = call(tool, args, db)
if result:
    print(json.dumps(result, indent=2))
"
}

clear
echo "=== Sift: Flaky Test Detection ==="
echo "Detecting tests that intermittently pass and fail across CI runs"
echo ""
pause 2

echo "# Run 1: CI pipeline - Monday morning"
type_cmd "# 20 tests, 5 failures"
pause 1

python3 -c "
import base64, json, os, subprocess

with open('testdata/flaky-run-1.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'api-ci'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_flaky.db'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Ingested: {r[\"total_tests\"]} tests, {r[\"failed\"]} failed')
            for g in r['failure_groups']:
                print(f'  [{g[\"category\"].upper()}] {g[\"root_cause\"]}')
                print(f'    {g[\"affected_tests\"]} tests ({g[\"original_failure_count\"]} original)')
    except: pass
"
pause 2

echo ""
echo "# Run 2: CI pipeline - Monday afternoon (same suite, different failures)"
type_cmd "# Some tests recovered, some new failures appeared"
pause 1

python3 -c "
import base64, json, os, subprocess

with open('testdata/flaky-run-2.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'api-ci'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_flaky.db'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Ingested: {r[\"total_tests\"]} tests, {r[\"failed\"]} failed')
            for g in r['failure_groups']:
                print(f'  [{g[\"category\"].upper()}] {g[\"root_cause\"]}')
                print(f'    {g[\"affected_tests\"]} tests ({g[\"original_failure_count\"]} original)')
    except: pass
"
pause 2

echo ""
echo "# Now ask sift: which tests are flaky?"
type_cmd "mcp__sift__get_flaky_tests"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_flaky_tests','arguments':{'time_range':'7d','min_runs':2}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_flaky.db'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Reports analyzed: {r[\"reports_analyzed\"]}')
            print(f'  Flaky tests found: {r[\"flaky_count\"]}')
            print()
            for t in r.get('flaky_tests', []):
                name = t['name'].split('.')[-1]
                print(f'  {name}  ({t[\"classname\"].split(\".\")[-1]})')
                print(f'    Flakiness rate: {t[\"flakiness_rate\"]:.0f}%  |  Runs: {t[\"total_runs\"]}  |  Failures: {t[\"failures\"]}')
    except: pass
"
pause 3

echo ""
echo "# These tests pass sometimes and fail other times."
echo "# Flaky tests waste CI time and erode trust in the suite."
echo "# Sift surfaces them automatically across runs."
pause 3
