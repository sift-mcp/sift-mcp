#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

DB="/tmp/sift_demo_pytest.db"
rm -f "$DB"

type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.04
    done
    echo
}

type_slow() {
    local text="$1"
    for (( i=0; i<${#text}; i++ )); do
        printf '%s' "${text:$i:1}"
        sleep 0.02
    done
    echo
}

pause() { sleep "${1:-1.5}"; }

clear
echo "=== Sift: Works with Any Test Runner ==="
echo "pytest, Go test, Maven, Gradle - anything that outputs JUnit XML"
echo ""
pause 2

echo "# pytest produces JUnit XML with --junit-xml"
type_cmd "pytest tests/ --junit-xml=report.xml -q"
pause 1
echo ""
echo "  FAILED tests/test_auth.py::test_token_refresh"
echo "  FAILED tests/test_auth.py::test_token_expiry"
echo "  FAILED tests/test_api.py::test_get_products"
echo "  FAILED tests/test_api.py::test_create_product"
echo "  FAILED tests/test_api.py::test_search_products"
echo "  ERROR  tests/test_api.py::test_get_orders"
echo "  26 passed, 5 failed, 1 error in 10.40s"
pause 2

echo ""
echo "# Ingest into sift"
type_cmd "mcp__sift__ingest_report"
pause 1

python3 -c "
import base64, json, os, subprocess

with open('testdata/pytest-report.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'pytest-ci'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_pytest.db'
proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Total: {r[\"total_tests\"]}  Failed: {r[\"failed\"]}  Passed: {r[\"passed\"]}')
            print()
            for i, g in enumerate(r['failure_groups'], 1):
                print(f'  {i}. [{g[\"category\"].upper()}] {g[\"root_cause\"]}')
                print(f'     {g[\"affected_tests\"]} tests ({g[\"original_failure_count\"]} original, {g[\"cascade_failure_count\"]} cascade)')
            print()
            print(f'  Summary: {r[\"summary\"]}')
    except: pass
"
pause 3

echo ""
echo "# Same pipeline. Same structured output."
echo "# Language does not matter - sift reads the XML, not the runner."
pause 3
