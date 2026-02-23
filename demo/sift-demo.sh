#!/usr/bin/env bash
set -e

# Typing simulation
type_cmd() {
    local cmd="$1"
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.04
    done
    echo
    eval "$cmd"
}

pause() { sleep "${1:-1.5}"; }

clear
echo "=== Sift: Test Intelligence MCP Server ==="
echo "Root Cause Extraction + Cascade Failure Detection"
echo ""
pause 2

echo "# Build the binary"
type_cmd "go build -o sift ./cmd/server"
pause 1

echo ""
echo "# Peek at our test report - cBioPortal (real-world Java project)"
type_cmd "head -5 testdata/cbioportal-report.xml"
pause 2

echo ""
echo "# Ingest and analyze through the MCP pipeline"
echo "# Pipeline: Extract -> RootCauseExtract -> Fingerprint -> Enrich -> CascadeDetect -> Summarize"
pause 2

type_cmd "python3 demo/ingest_and_display.py"
pause 3

echo ""
echo "# Key takeaway:"
echo "#   - 12 original failures, 84 cascading (87.5% was noise)"
echo "#   - 3 clean root causes instead of 220 raw errors"
echo "#   - Complete sentences, not truncated framework dumps"
pause 3
