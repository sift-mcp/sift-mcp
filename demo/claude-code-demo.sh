#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.03
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
echo "=== Connecting Sift to Claude Code ==="
echo ""
pause 2

echo "Step 1: Add Sift as an MCP server"
echo ""
type_cmd "claude mcp add sift-mcp -- npx sift-mcp"
pause 1
echo "  Done. Sift is now available inside Claude Code."
echo ""
pause 2

echo "Step 2: Restart Claude Code - tools appear automatically"
echo ""
echo "  Claude Code now has 6 new tools:"
echo "    - mcp__sift__ingest_report"
echo "    - mcp__sift__analyze_results"
echo "    - mcp__sift__get_report_stats"
echo "    - mcp__sift__get_failure_history"
echo "    - mcp__sift__get_flaky_tests"
echo "    - mcp__sift__get_severity_trend"
echo ""
pause 3

echo "Step 3: Use it naturally"
echo ""
echo '  You say: "Run my tests and analyze the failures with Sift"'
echo ""
pause 2
echo "  Claude Code will:"
echo "    1. Run your test suite (pytest, go test, mvn test, etc.)"
echo "    2. Encode the JUnit XML report"
echo "    3. Call mcp__sift__ingest_report"
echo "    4. Read the structured analysis"
echo "    5. Give you actionable suggestions based on root causes"
echo ""
pause 3

echo "--- Example: What Claude Code sees after ingestion ---"
echo ""
pause 1

# Actually run the ingestion so the output is real
python3 -c "
import base64, json, os, subprocess

with open('testdata/cbioportal-report.xml', 'rb') as f:
    b64 = base64.b64encode(f.read()).decode()

init = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'protocolVersion':'2024-11-05','capabilities':{},'clientInfo':{'name':'demo','version':'1.0'}}})
call = json.dumps({'jsonrpc':'2.0','id':2,'method':'tools/call','params':{'name':'ingest_report','arguments':{'report_xml_base64':b64,'source':'cbioportal-ci'}}})

env = os.environ.copy()
env['SIFT_DB_PATH'] = '/tmp/sift_claude_demo.db'
if os.path.exists(env['SIFT_DB_PATH']):
    os.remove(env['SIFT_DB_PATH'])

proc = subprocess.run(['./sift'], input=init+'\n'+call+'\n', capture_output=True, text=True, timeout=30, env=env)

for line in proc.stdout.strip().split('\n'):
    try:
        obj = json.loads(line)
        if obj.get('id') == 2:
            r = json.loads(obj['result']['content'][0]['text'])
            print(f'  Summary: {r[\"summary\"]}')
            print()
            for i, g in enumerate(r['failure_groups'], 1):
                print(f'  Group {i}: {g[\"root_cause\"]}')
                print(f'           [{g[\"category\"]}] {g[\"affected_tests\"]} tests ({g[\"original_failure_count\"]} original, {g[\"cascade_failure_count\"]} cascade)')
            print()

            cs = r.get('cascade_summary')
            if cs:
                print(f'  Cascade detection: {cs[\"cascade_percentage\"]:.0f}% of failures were cascading noise')
                print(f'  Only {cs[\"total_original_failures\"]} root failures to investigate (not {r[\"failed\"]})')
    except: pass
"
pause 3

echo ""
echo "--- What Claude Code tells you ---"
echo ""
type_slow '  "220 tests failed but there are only 2 root causes.'
type_slow '   88% of failures are cascading from the originals.'
type_slow ''
type_slow '   1. Docker is not running. The Testcontainers-based'
type_slow '      Clickhouse tests need Docker. Start Docker or skip'
type_slow '      these tests with: mvn test -pl !clickhouse-module'
type_slow ''
type_slow '   2. DataSource URL not configured. The MyBatis repository'
type_slow '      tests need a database connection. Check your'
type_slow '      application-test.properties for the JDBC URL."'
echo ""
pause 3

echo "=== That is Sift + Claude Code ==="
echo "Structured analysis, not raw logs."
echo ""
