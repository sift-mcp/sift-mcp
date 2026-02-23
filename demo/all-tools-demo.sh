#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

DB="/tmp/sift_demo_alltools.db"
rm -f "$DB"

type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.03
    done
    echo
    eval "$cmd"
}

pause() { sleep "${1:-1.5}"; }

clear
echo "=== Sift MCP Tools Demo ==="
echo "All 6 tools with a real cBioPortal test report"
echo ""
pause 2

# ---- Tool 1: ingest_report ----
echo "--- Tool 1: ingest_report ---"
echo "# Ingest cBioPortal JUnit XML (308 tests, 220 failures)"
pause 1

python3 -c "
import base64, json, os, subprocess

with open('testdata/cbioportal-report.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'cbioportal-ci'}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            result = json.loads(obj['result']['content'][0]['text'])

            # Save report_id for later tools
            with open('/tmp/sift_demo_report_id.txt', 'w') as f:
                f.write(result['report_id'])

            print(f\"\"\"  Report ID:   {result['report_id'][:8]}...
  Total tests: {result['total_tests']}
  Failed:      {result['failed']}
  Passed:      {result['passed']}

  Failure Groups ({len(result['failure_groups'])}):
\"\"\")
            for i, g in enumerate(result['failure_groups'], 1):
                print(f'    {i}. [{g[\"category\"].upper()}] {g[\"root_cause\"]}')
                print(f'       {g[\"affected_tests\"]} tests ({g[\"original_failure_count\"]} original, {g[\"cascade_failure_count\"]} cascade)')
                print()

            cs = result.get('cascade_summary')
            if cs:
                print(f'  Cascade: {cs[\"total_original_failures\"]} original, {cs[\"total_cascade_failures\"]} cascading ({cs[\"cascade_percentage\"]:.1f}% noise)')
                print()
            print(f'  Summary: {result[\"summary\"]}')
    except: pass
"
pause 3

# ---- Tool 2: get_report_stats ----
echo ""
echo "--- Tool 2: get_report_stats ---"
echo "# Aggregate statistics across all stored reports"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_report_stats','arguments':{'time_range':'30d'}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f\"\"\"  Reports stored:  {r['total_reports_all_time']}
  Pass rate:       {r['pass_rate_pct']}%
  Total tests:     {r['total_tests']}
  Total failed:    {r['total_failed']}
  Total errored:   {r['total_errored']}

  Top failing tests:\"\"\")
            for t in r.get('top_failing_tests', [])[:5]:
                name = t['name'].split('.')[-1] if '.' in t['name'] else t['name']
                print(f'    - {name} ({t[\"failure_count\"]}x)')
    except: pass
"
pause 3

# ---- Tool 3: analyze_results ----
echo ""
echo "--- Tool 3: analyze_results ---"
echo "# Re-analyze a previously stored report"
pause 1

REPORT_ID=$(cat /tmp/sift_demo_report_id.txt 2>/dev/null || echo "")
if [ -n "$REPORT_ID" ]; then
python3 -c "
import json, os, subprocess

report_id = '$REPORT_ID'
init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'analyze_results','arguments':{'report_id': report_id}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Re-analyzed report {report_id[:8]}...')
            print(f'  Failure groups: {len(r[\"failure_groups\"])}')
            print(f'  Summary: {r[\"summary\"]}')
    except: pass
"
else
    echo "  (skipped - no report_id available)"
fi
pause 2

# ---- Tool 4: get_failure_history ----
echo ""
echo "--- Tool 4: get_failure_history ---"
echo "# Check how many times a specific test has failed"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_failure_history','arguments':{'test_name':'getMetaMutationsInMultipleMolecularProfiles','limit':5}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Test: {r[\"test_name\"]}')
            print(f'  Failure count: {r[\"failure_count\"]}')
            for h in r.get('history', []):
                print(f'    - Report {h[\"report_id\"][:8]}... ({h[\"failed\"]}/{h[\"total_tests\"]} failed)')
    except: pass
"
pause 2

# ---- Tool 5: get_flaky_tests ----
echo ""
echo "--- Tool 5: get_flaky_tests ---"
echo "# Find tests that intermittently pass and fail"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_flaky_tests','arguments':{'time_range':'30d','min_runs':1}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Reports analyzed: {r[\"reports_analyzed\"]}')
            print(f'  Flaky tests found: {r[\"flaky_count\"]}')
            if r['flaky_count'] == 0:
                print('  (Need multiple reports to detect flakiness)')
            for t in r.get('flaky_tests', [])[:5]:
                print(f'    - {t[\"name\"]} ({t[\"flakiness_rate\"]:.0f}% flaky, {t[\"total_runs\"]} runs)')
    except: pass
"
pause 2

# ---- Tool 6: get_severity_trend ----
echo ""
echo "--- Tool 6: get_severity_trend ---"
echo "# Track failure severity distribution over time"
pause 1

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_severity_trend','arguments':{'time_range':'30d','bucket':'day'}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '$DB'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Time range: {r[\"time_range\"]}')
            print(f'  Bucket: {r[\"bucket\"]}')
            trend = r.get('trend', {})
            if not trend:
                print('  (No trend data yet)')
            for date, severities in sorted(trend.items()):
                parts = [f'{sev}: {count}' for sev, count in sorted(severities.items())]
                print(f'    {date}: {\"  \".join(parts)}')
    except: pass
"
pause 2

echo ""
echo "=== All 6 tools demonstrated ==="
echo ""
echo "Tools: ingest_report, get_report_stats, analyze_results,"
echo "       get_failure_history, get_flaky_tests, get_severity_trend"
echo ""
pause 2
