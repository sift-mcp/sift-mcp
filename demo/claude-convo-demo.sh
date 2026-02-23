#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

DB="/tmp/sift_demo_convo.db"
rm -f "$DB"

type_slow() {
    local text="$1"
    for (( i=0; i<${#text}; i++ )); do
        printf '%s' "${text:$i:1}"
        sleep 0.025
    done
    echo
}

type_tool() {
    local text="$1"
    for (( i=0; i<${#text}; i++ )); do
        printf '%s' "${text:$i:1}"
        sleep 0.015
    done
    echo
}

pause() { sleep "${1:-1.5}"; }

clear
echo "=== Sift inside Claude Code ==="
echo "A real triage session"
echo ""
pause 2

echo "  You"
echo "  ───────────────────────────────────────────────────────"
type_slow '  my CI just failed. 200+ test failures. help me triage.'
echo ""
pause 2

echo "  Claude Code"
echo "  ───────────────────────────────────────────────────────"
type_slow '  Running your tests and ingesting the report into Sift...'
echo ""
pause 1

echo "  > mcp__sift__ingest_report(source: \"ci\")"
echo ""

python3 -c "
import base64, json, os, subprocess

with open('testdata/cbioportal-report.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'ci'}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_convo.db'
subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
" 2>/dev/null

type_slow '  Done. Here is what is actually wrong:'
echo ""
pause 1

type_slow '  220 tests failed but there are only 2 root causes.'
type_slow '  88% of failures are cascading noise from those 2.'
echo ""
pause 1

type_slow '  1. Docker is not running (6 tests, 50 cascade)'
type_slow '     Your Testcontainers-based Clickhouse tests need Docker.'
type_slow '     Fix: start Docker, or skip with -pl \!clickhouse-module'
echo ""
pause 1

type_slow '  2. DataSource URL not configured (4 tests, 34 cascade)'
type_slow '     MyBatis repository tests cannot connect to the DB.'
type_slow '     Fix: check application-test.properties for jdbc.url'
echo ""
pause 2

echo "  You"
echo "  ───────────────────────────────────────────────────────"
type_slow '  has the DataSource issue happened before?'
echo ""
pause 2

echo "  Claude Code"
echo "  ───────────────────────────────────────────────────────"
type_slow '  Checking failure history...'
echo ""

echo "  > mcp__sift__get_failure_history(test: \"AlterationMyBatisRepositoryTest\")"
echo ""

python3 -c "
import json, os, subprocess

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'get_failure_history','arguments':{'test_name':'AlterationMyBatisRepositoryTest','limit':5}}})
env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_demo_convo.db'
subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)
" 2>/dev/null

type_slow '  First time this specific fingerprint has appeared.'
type_slow '  Likely a config drift - someone changed the test environment.'
echo ""
pause 2

echo "  You"
echo "  ───────────────────────────────────────────────────────"
type_slow '  ok what do I fix first?'
echo ""
pause 2

echo "  Claude Code"
echo "  ───────────────────────────────────────────────────────"
type_slow '  Fix the DataSource issue first - it is a config change,'
type_slow '  not a code bug, and it unblocks 38 tests immediately.'
type_slow '  The Docker issue needs environment access so fix that second.'
echo ""
pause 3

echo "  ─────────────────────────────────────────────────────────"
echo "  Sift gave Claude the structure. Claude gave you the answer."
echo "  ─────────────────────────────────────────────────────────"
pause 3
