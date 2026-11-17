# ibox — local dev orchestration
# Prerequisites: go 1.23+, node 20+, lima (macOS)

VM      := ibox
CONTROL_LOG := /tmp/ibox-control.log
AGENT_LOG   := /tmp/ibox-agent.log

.PHONY: all build vm vm-up agent control web test e2e clean

all: build

build:
	go build ./...
	cd web && npm install && npm run build

# --- execution plane (Lima VM) ---

vm:
	limactl start --name=$(VM) deploy/ibox-vm.yaml

vm-up:
	limactl start $(VM)

vm-provision: /tmp/ibox-agent
	limactl copy /tmp/ibox-agent $(VM):/tmp/ibox-agent.new
	limactl shell $(VM) -- sudo install -m 0755 /tmp/ibox-agent.new /usr/local/bin/ibox-agent

/tmp/ibox-agent:
	GOOS=linux GOARCH=arm64 go build -o /tmp/ibox-agent ./cmd/agent

agent: /tmp/ibox-agent vm-provision
	limactl shell $(VM) -- sudo bash -c 'pkill -x ibox-agent; sleep 1; setsid nohup /usr/local/bin/ibox-agent \
	  -control ws://host.lima.internal:8080/internal/nodes/connect \
	  -node-id node-1 -token $${IBOX_NODE_TOKEN:-dev-node-token} \
	  >> /var/log/ibox-agent.log 2>&1 < /dev/null & sleep 2; pgrep -x ibox-agent'

# --- control plane ---

control:
	IBOX_NODE_TOKEN=$${IBOX_NODE_TOKEN:-dev-node-token} go run ./cmd/control \
	  -db $${IBOX_DB:-/tmp/ibox.db} -addr :8080 -proxy-addr :8081

# --- web dashboard ---

web:
	cd web && npm install && npm run dev

# --- end-to-end test (requires control plane + agent running) ---

e2e:
	./scripts/e2e.sh

clean:
	pkill -x control 2>/dev/null || true
	limactl shell $(VM) -- sudo pkill -x ibox-agent 2>/dev/null || true
	rm -f /tmp/ibox.db*
