#!/usr/bin/env python3
import base64, json, os, subprocess, sys, time

SIFT_BINARY = "./sift"
DB_PATH = "/tmp/sift_demo.db"

def call_mcp(tool_name, arguments, db_path=DB_PATH):
    init_msg = json.dumps({
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "sift-demo", "version": "1.0"}
        }
    })
    call_msg = json.dumps({
        "jsonrpc": "2.0", "id": 2, "method": "tools/call",
        "params": {"name": tool_name, "arguments": arguments}
    })

    env = os.environ.copy()
    env["SIFT_DB_PATH"] = db_path

    proc = subprocess.run(
        [SIFT_BINARY],
        input=init_msg + "\n" + call_msg + "\n",
        capture_output=True, text=True, timeout=30, env=env
    )

    for line in proc.stdout.strip().split("\n"):
        try:
            obj = json.loads(line)
            if obj.get("id") == 2:
                content = obj.get("result", {}).get("content", [])
                is_error = obj.get("result", {}).get("isError", False)
                for c in content:
                    try:
                        return json.loads(c["text"]), is_error
                    except json.JSONDecodeError:
                        return c["text"], is_error
        except json.JSONDecodeError:
            pass
    return None, True

def print_json(data, indent=2):
    print(json.dumps(data, indent=indent, default=str))

def print_header(title):
    print(f"\n{'='*60}")
    print(f"  {title}")
    print(f"{'='*60}\n")

def print_tool_call(tool_name, args_summary=""):
    label = f"mcp__sift__{tool_name}"
    if args_summary:
        label += f"({args_summary})"
    print(f"  >>> {label}")
    print()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 demo/mcp_client.py <tool_name> [args_json]")
        print("Tools: ingest_report, analyze_results, get_report_stats,")
        print("       get_failure_history, get_flaky_tests, get_severity_trend")
        sys.exit(1)

    tool = sys.argv[1]
    args = {}
    if len(sys.argv) > 2:
        args = json.loads(sys.argv[2])

    result, is_error = call_mcp(tool, args)
    if is_error:
        print(f"Error: {result}", file=sys.stderr)
        sys.exit(1)
    print_json(result)
