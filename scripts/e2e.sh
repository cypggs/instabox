#!/usr/bin/env bash
# ibox end-to-end test (Phase 5): full machine lifecycle against a live deployment.
# Prerequisites: control plane on :8080, agent connected, web optional.
# Usage: IBOX_BASE=http://localhost:8080 IBOX_EMAIL=admin@ibox.local IBOX_PASSWORD=admin1234 ./scripts/e2e.sh
set -u

BASE="${IBOX_BASE:-http://localhost:8080}"
EMAIL="${IBOX_EMAIL:-admin@ibox.local}"
PASS="${IBOX_PASSWORD:-admin1234}"
COOKIE_JAR="$(mktemp)"
trap 'rm -f "$COOKIE_JAR"' EXIT

PASS_COUNT=0
FAIL_COUNT=0
CURRENT_STEP=""
CSRF=""

step() { CURRENT_STEP="$1"; printf '\n\033[1m[%s]\033[0m %s\n' "$(date +%H:%M:%S)" "$1"; }
ok()   { PASS_COUNT=$((PASS_COUNT+1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
fail() { FAIL_COUNT=$((FAIL_COUNT+1)); printf '  \033[31mFAIL\033[0m %s\n' "$1"; }
check() { # check <description> <actual> <expected-substring>
  if [[ "$2" == *"$3"* ]]; then ok "$1"; else fail "$1 — expected [$3], got [$2]"; fi
}

jqt() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval(sys.argv[1]))" "$1" 2>/dev/null; }

req() { # req <METHOD> <path> [json-body] -> stdout body ; sets HTTP_CODE
  local method="$1" path="$2" body="${3:-}"
  local args=(-s -b "$COOKIE_JAR" -c "$COOKIE_JAR" -X "$method" "$BASE$path" -H "content-type: application/json" -w $'\n%{http_code}')
  [[ -n "$CSRF" && "$method" != "GET" ]] && args+=(-H "x-ibox-csrf: $CSRF")
  [[ -n "$body" ]] && args+=(-d "$body")
  local out
  out=$(curl "${args[@]}")
  HTTP_CODE=$(tail -n1 <<<"$out")
  sed '$d' <<<"$out"
}

# ---------- 1. login ----------
step "1. login and obtain session + CSRF"
LOGIN=$(req POST /v1/auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
CSRF=$(echo "$LOGIN" | jqt "d.get('csrfToken','')")
check "login ok" "$LOGIN" '"ok":true'
[[ -z "$CSRF" ]] && { fail "no csrf token"; exit 1; }
ok "csrf token present"

# ---------- 2. create machine ----------
step "2. create machine (async, 202)"
# clean up snapshots left by previous runs (free-plan quota is 5)
for SID in $(req GET /v1/snapshots | python3 -c "import sys,json
try:
  print(' '.join(s['id'] for s in json.load(sys.stdin).get('snapshots',[])))
except Exception:
  pass"); do
  req DELETE "/v1/snapshots/$SID" >/dev/null
done
ok "snapshot quota cleaned"
MACHINE_ID=""
CREATE=$(req POST /v1/machines '{"name":"e2e-lifecycle","templateId":"minimal","vcpus":4,"memMiB":8192,"persistentDiskGiB":2,"env":{"E2E_MARKER":"hello-ibox"},"autoSuspendAfterSeconds":1800}')
MACHINE_ID=$(echo "$CREATE" | jqt "d.get('id','')")
check "create accepted (machine id)" "$MACHINE_ID" "m_"
check "initial state creating" "$(echo "$CREATE" | jqt "d.get('state','')")" "creating"

# duplicate name must be rejected
DUP=$(req POST /v1/machines '{"name":"e2e-lifecycle","templateId":"minimal","vcpus":4,"memMiB":8192}')
check "duplicate name rejected" "$DUP" "already_exists"

# ---------- 3. wait until running ----------
step "3. wait for state=running"
DEADLINE=$(( $(date +%s) + 900 ))
STATE=""
while (( $(date +%s) < DEADLINE )); do
  M=$(req GET "/v1/machines/$MACHINE_ID")
  STATE=$(echo "$M" | jqt "d.get('state','')")
  [[ "$STATE" == "running" ]] && break
  [[ "$STATE" == "error" ]] && { fail "machine went to error: $M"; exit 1; }
  sleep 10
done
check "state running" "$STATE" "running"

# ---------- 4. env var present ----------
step "4. env var visible inside sandbox"
EXEC=$(req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"echo marker=$E2E_MARKER; whoami; hostname","timeoutSeconds":30}')
check "env marker" "$(echo "$EXEC" | jqt "d.get('stdout','')")" "marker=hello-ibox"
check "guest user" "$(echo "$EXEC" | jqt "d.get('stdout','')")" "user"
check "hostname is machine name" "$(echo "$EXEC" | jqt "d.get('stdout','')")" "e2e-lifecycle"
check "exit code 0" "$(echo "$EXEC" | jqt "d.get('exitCode')")" "0"

# ---------- 5. write + read file under /data ----------
step "5. write and read file via files API"
HTTP_CODE=""
WR=$(curl -s -b "$COOKIE_JAR" -X PUT "$BASE/v1/machines/$MACHINE_ID/files?path=/data/e2e/hello.txt&parents=true" \
  -H "x-ibox-csrf: $CSRF" -H "content-type: application/octet-stream" -d "hello-ibox-$(date +%s)" -w $'\n%{http_code}')
check "file write (204)" "$(tail -n1 <<<"$WR")" "204"
RD=$(curl -s -b "$COOKIE_JAR" "$BASE/v1/machines/$MACHINE_ID/files/content?path=/data/e2e/hello.txt")
check "file read back" "$RD" "hello-ibox-"

# ---------- 6. list directory ----------
step "6. list /data directory"
LS=$(req GET "/v1/machines/$MACHINE_ID/files?path=/data/e2e")
check "dir listing contains file" "$(echo "$LS" | jqt "[e['name'] for e in d.get('entries',[])]")" "hello.txt"

# ---------- 7. exec sees the file ----------
step "7. exec sees file written via files API"
EXEC2=$(req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"cat /data/e2e/hello.txt","timeoutSeconds":30}')
check "exec reads file" "$(echo "$EXEC2" | jqt "d.get('stdout','')")" "hello-ibox-"

# ---------- 8. terminal websocket ----------
step "8. terminal websocket attach + command roundtrip"
TERM_OUT=$(python3 - "$BASE" "$MACHINE_ID" "$COOKIE_JAR" <<'PYEOF'
import sys, json, base64, time, urllib.request, socket, hashlib, os, struct, random
base, mid, jar = sys.argv[1], sys.argv[2], sys.argv[3]
# read session cookie
cookie = ""
for line in open(jar):
    p = line.split()
    if len(p) >= 7 and p[5] == "ibox_session":
        cookie = p[6]
url = base.replace("http", "ws") + f"/v1/machines/{mid}/terminal"
hostport = base.split("//")[1]
host, port = hostport.split(":")
key = base64.b64encode(os.urandom(16)).decode()
s = socket.create_connection((host, int(port)), timeout=20)
req = (f"GET /v1/machines/{mid}/terminal HTTP/1.1\r\nHost: {hostport}\r\n"
       f"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\n"
       f"Sec-WebSocket-Version: 13\r\nCookie: ibox_session={cookie}\r\n\r\n")
s.sendall(req.encode())
buf = b""
while b"\r\n\r\n" not in buf:
    buf += s.recv(4096)
if b"101" not in buf.split(b"\r\n")[0]:
    print("HANDSHAKE_FAILED"); sys.exit(0)

def ws_send(data: bytes):
    mask = os.urandom(4)
    header = b"\x81"
    n = len(data)
    if n < 126: header += bytes([0x80 | n])
    elif n < 65536: header += bytes([0x80 | 126]) + struct.pack(">H", n)
    else: header += bytes([0x80 | 127]) + struct.pack(">Q", n)
    masked = bytes(b ^ mask[i % 4] for i, b in enumerate(data))
    s.sendall(header + mask + masked)

def ws_recv_frame():
    hdr = s.recv(2)
    if len(hdr) < 2: return None
    ln = hdr[1] & 0x7F
    if ln == 126: ln = struct.unpack(">H", s.recv(2))[0]
    elif ln == 127: ln = struct.unpack(">Q", s.recv(8))[0]
    data = b""
    while len(data) < ln:
        chunk = s.recv(ln - len(data))
        if not chunk: break
        data += chunk
    return data

ws_send(json.dumps({"cols": 120, "rows": 30}).encode())
time.sleep(1.0)
ws_send(b"echo term-$((40+2))-ok\n")
out = b""
s.settimeout(25)
end = time.time() + 22
while time.time() < end:
    try:
        fr = ws_recv_frame()
    except socket.timeout:
        break
    if fr is None: break
    out += fr
    if b"term-42-ok" in out:
        break
print(out.decode(errors="replace"))
PYEOF
)
check "terminal roundtrip" "$TERM_OUT" "term-42-ok"

# ---------- 9. cpu/mem metrics ----------
step "9. cpu/memory visible inside sandbox + metrics API"
BURN=$(req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"sh -c \"for i in 1 2 3; do (dd if=/dev/zero of=/dev/null bs=1M count=200 &); done\"; sleep 2; grep -E \"MemTotal|MemAvailable\" /proc/meminfo; nproc","timeoutSeconds":60}')
check "meminfo present" "$BURN" "MemTotal"
check "nproc = 4 vcpus" "$(echo "$BURN" | jqt "d.get('stdout','')")" "4"
sleep 35
METRICS=$(req GET "/v1/machines/$MACHINE_ID/metrics?minutes=5")
NP=$(echo "$METRICS" | jqt "len(d.get('points',[]))")
if (( NP > 0 )); then ok "metrics samples present ($NP)"; else fail "no metric samples"; fi

# ---------- 10. stop ----------
step "10. stop machine"
req POST "/v1/machines/$MACHINE_ID/stop" '{}' >/dev/null
DEADLINE=$(( $(date +%s) + 120 ))
while (( $(date +%s) < DEADLINE )); do
  STATE=$(req GET "/v1/machines/$MACHINE_ID" | jqt "d.get('state','')")
  [[ "$STATE" == "stopped" ]] && break
  sleep 5
done
check "state stopped" "$STATE" "stopped"

# ---------- 11. start again ----------
step "11. start machine again"
req POST "/v1/machines/$MACHINE_ID/start" '{}' >/dev/null
DEADLINE=$(( $(date +%s) + 180 ))
while (( $(date +%s) < DEADLINE )); do
  M=$(req GET "/v1/machines/$MACHINE_ID")
  STATE=$(echo "$M" | jqt "d.get('state','')")
  [[ "$STATE" == "running" ]] && break
  sleep 5
done
check "state running again" "$STATE" "running"

# ---------- 12. /data persisted across restart ----------
step "12. /data file survived stop/start"
EXEC3=$(req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"cat /data/e2e/hello.txt","timeoutSeconds":30}')
check "file persisted" "$(echo "$EXEC3" | jqt "d.get('stdout','')")" "hello-ibox-"

# ---------- 13. snapshot + restore ----------
step "13. live snapshot, modify, restore, verify rollback"
SNAP=$(req POST "/v1/machines/$MACHINE_ID/snapshots" '{"name":"e2e-snap"}')
SNAP_ID=$(echo "$SNAP" | jqt "d.get('id','')")
check "snapshot created" "$SNAP_ID" "sn_"
sleep 8
req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"echo changed > /data/e2e/hello.txt","timeoutSeconds":30}' >/dev/null
REST=$(req POST "/v1/machines/$MACHINE_ID/restore" "{\"snapshotId\":\"$SNAP_ID\",\"desiredState\":\"stopped\"}")
check "restore accepted" "$REST" '"ok":true'
sleep 20
req POST "/v1/machines/$MACHINE_ID/start" '{}' >/dev/null
DEADLINE=$(( $(date +%s) + 180 ))
while (( $(date +%s) < DEADLINE )); do
  STATE=$(req GET "/v1/machines/$MACHINE_ID" | jqt "d.get('state','')")
  [[ "$STATE" == "running" ]] && break
  sleep 5
done
EXEC4=$(req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"cat /data/e2e/hello.txt","timeoutSeconds":30}')
check "restored content (rollback)" "$(echo "$EXEC4" | jqt "d.get('stdout','')")" "hello-ibox-"

# ---------- 14. pause (suspend) + resume ----------
step "14. suspend + resume preserves processes"
req POST "/v1/machines/$MACHINE_ID/exec" '{"cmd":"nohup sh -c \"sleep 600; echo done > /data/e2e/sleeper.txt\" > /dev/null 2>&1 & echo $!","timeoutSeconds":30}' >/dev/null
sleep 2
req POST "/v1/machines/$MACHINE_ID/suspend" '{}' >/dev/null
DEADLINE=$(( $(date +%s) + 180 ))
while (( $(date +%s) < DEADLINE )); do
  M=$(req GET "/v1/machines/$MACHINE_ID")
  STATE=$(echo "$M" | jqt "d.get('state','')")
  PRES=$(echo "$M" | jqt "d.get('runtime',{}).get('processesPreserved',False)")
  [[ "$STATE" == "suspended" ]] && break
  sleep 5
done
check "state suspended" "$STATE" "suspended"
if [[ "$(echo "$M" | jqt "d.get('runtime',{}).get('processesPreserved',False)")" == "True" ]]; then
  ok "processesPreserved=true"
else
  fail "processesPreserved expected true"
fi
req POST "/v1/machines/$MACHINE_ID/resume" '{}' >/dev/null
DEADLINE=$(( $(date +%s) + 180 ))
while (( $(date +%s) < DEADLINE )); do
  STATE=$(req GET "/v1/machines/$MACHINE_ID" | jqt "d.get('state','')")
  [[ "$STATE" == "running" ]] && break
  sleep 5
done
check "resumed to running" "$STATE" "running"

# ---------- 15. delete + verify cleanup ----------
step "15. delete machine and verify cleanup"
DEL=$(curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/v1/machines/$MACHINE_ID" -H "x-ibox-csrf: $CSRF" -H "content-type: application/json" -d "{\"name\":\"e2e-lifecycle\"}" -w $'\n%{http_code}')
check "delete accepted (202)" "$(tail -n1 <<<"$DEL")" "202"
DEADLINE=$(( $(date +%s) + 180 ))
while (( $(date +%s) < DEADLINE )); do
  GONE=$(req GET "/v1/machines/$MACHINE_ID")
  if echo "$GONE" | grep -q "not_found"; then break; fi
  sleep 5
done
check "machine gone" "$GONE" "not_found"
LIST=$(req GET /v1/machines)
check "not in machine list" "$(echo "$LIST" | jqt "[m['id'] for m in d.get('machines',[])]")" "[]"

# ---------- summary ----------
printf '\n\033[1mE2E RESULT\033[0m: %d passed, %d failed\n' "$PASS_COUNT" "$FAIL_COUNT"
exit $(( FAIL_COUNT > 0 ? 1 : 0 ))
