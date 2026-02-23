#!/usr/bin/env python3
import base64, json, subprocess, os, textwrap

with open("testdata/cbioportal-report.xml", "rb") as f:
    b64 = base64.b64encode(f.read()).decode()

init_msg = json.dumps({
    "jsonrpc": "2.0", "id": 1, "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "demo", "version": "1.0"}
    }
})

call_msg = json.dumps({
    "jsonrpc": "2.0", "id": 2, "method": "tools/call",
    "params": {
        "name": "ingest_report",
        "arguments": {"report_xml_base64": b64, "source": "cbioportal-ci"}
    }
})

env = os.environ.copy()
env["SIFT_DB_PATH"] = "/tmp/sift_demo.db"

# Clean slate
if os.path.exists("/tmp/sift_demo.db"):
    os.remove("/tmp/sift_demo.db")

proc = subprocess.run(
    ["./sift"],
    input=init_msg + "\n" + call_msg + "\n",
    capture_output=True, text=True, timeout=30, env=env
)

for line in proc.stdout.strip().split("\n"):
    try:
        obj = json.loads(line)
        if obj.get("id") == 2:
            content = obj["result"]["content"]
            for c in content:
                result = json.loads(c["text"])

                print("\n--- Analysis Result ---\n")
                print(f"Total tests:  {result['total_tests']}")
                print(f"Failed:       {result['failed']}")
                print(f"Passed:       {result['passed']}")

                print(f"\n--- Failure Groups ({len(result['failure_groups'])}) ---\n")
                for i, g in enumerate(result["failure_groups"], 1):
                    print(f"  {i}. [{g['category'].upper()}] {g['root_cause']}")
                    print(f"     Affected: {g['affected_tests']} tests "
                          f"({g['original_failure_count']} original, "
                          f"{g['cascade_failure_count']} cascade)")
                    suites = ", ".join(g["affected_suites"][:4])
                    if len(g["affected_suites"]) > 4:
                        suites += f" +{len(g['affected_suites']) - 4} more"
                    print(f"     Suites: {suites}")
                    print()

                cs = result.get("cascade_summary")
                if cs:
                    print("--- Cascade Summary ---\n")
                    print(f"  Original failures:  {cs['total_original_failures']}")
                    print(f"  Cascade failures:   {cs['total_cascade_failures']}")
                    print(f"  Cascade percentage: {cs['cascade_percentage']:.1f}%")
                    print()

                print("--- Summary ---\n")
                print(f"  {result['summary']}")
                print()
    except (json.JSONDecodeError, KeyError):
        pass
