#!/usr/bin/env bash
#
# Re-record all demo GIFs with properly sized terminals.
# Usage: ./demo/record-all.sh
#
# Prerequisites:
#   - asciinema (pip install asciinema)
#   - agg       (cargo install agg)
#   - ./sift binary built (go build -o sift ./cmd/server)
#
set -e
cd "$(dirname "$0")/.."

DEMO_DIR="demo"
COLS=100

# Demo name -> rows (sized to fit content tightly with a small margin)
declare -A DEMOS
DEMOS=(
  ["sift-demo"]=30
  ["all-tools-demo"]=38
  ["claude-code-demo"]=38
  ["flaky-demo"]=32
  ["trend-demo"]=30
  ["pytest-demo"]=28
  ["claude-convo-demo"]=38
)

record_demo() {
  local name="$1"
  local rows="$2"
  local script="$DEMO_DIR/${name}.sh"
  local cast="$DEMO_DIR/${name}.cast"
  local gif="$DEMO_DIR/${name}.gif"

  if [ ! -f "$script" ]; then
    echo "SKIP: $script not found"
    return
  fi

  echo ""
  echo "=== Recording: $name (${COLS}x${rows}) ==="

  # Record with asciinema at the exact terminal size
  ASCIINEMA_REC=1 asciinema rec \
    --cols "$COLS" \
    --rows "$rows" \
    --overwrite \
    --command "bash $script" \
    "$cast"

  echo "  -> $cast"

  # Convert to GIF with agg
  agg --cols "$COLS" --rows "$rows" --font-size 14 "$cast" "$gif"

  echo "  -> $gif"

  # Show resulting dimensions
  if command -v identify &>/dev/null; then
    dims=$(identify "$gif" 2>/dev/null | head -1 | grep -oP '\d+x\d+' | head -1)
    echo "  -> GIF dimensions: $dims"
  fi
}

echo "Recording all demos..."
echo "Columns: $COLS"
echo ""

for name in sift-demo all-tools-demo claude-code-demo flaky-demo trend-demo pytest-demo claude-convo-demo; do
  record_demo "$name" "${DEMOS[$name]}"
done

echo ""
echo "Done! All GIFs regenerated in $DEMO_DIR/"
echo ""
echo "Check them with:"
echo "  ls -lh $DEMO_DIR/*.gif"
echo "  identify $DEMO_DIR/*.gif 2>/dev/null | head -7"
