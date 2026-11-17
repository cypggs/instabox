// Command agent runs the ibox execution-plane node agent.
// It runs inside the Linux VM (Lima) and connects to the control plane.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibox/ibox/internal/agent/sandbox"
	agentserver "github.com/ibox/ibox/internal/agent/server"
)

func main() {
	var (
		control = flag.String("control", "ws://host.lima.internal:8080/internal/nodes/connect", "control plane WS URL")
		nodeID  = flag.String("node-id", "node-1", "node ID")
		token   = flag.String("token", env("IBOX_NODE_TOKEN", ""), "node auth token")
		baseDir = flag.String("base-dir", "/var/lib/ibox-agent", "agent state dir")
		runtime = flag.String("runtime", "runsc", "sandbox runtime: runsc|runc")
		network = flag.String("network", "host", "sandbox network: host|none")
	)
	flag.Parse()

	mgr := sandbox.NewManager(*baseDir, *runtime, *network)
	agent := agentserver.New(*control, *nodeID, *token, mgr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("ibox agent %s starting (runtime=%s network=%s)", *nodeID, *runtime, *network)
	if err := agent.Run(ctx); err != nil {
		log.Fatalf("agent: %v", err)
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
