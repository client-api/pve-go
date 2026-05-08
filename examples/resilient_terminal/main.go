// Example: resilient terminal session with auto-reconnect.
//
// Run with:
//
//	PVE_HOST=https://pve.example.com:8006 \
//	PVE_TOKEN='PVEAPIToken=root@pam!auto=...' \
//	PVE_NODE=orca PVE_VMID=100 \
//	go run ./examples/resilient_terminal
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	pve "github.com/client-api/pve-go"
)

func main() {
	host := os.Getenv("PVE_HOST")
	if host == "" {
		host = "https://localhost:8006"
	}
	cfg := pve.NewConfiguration()
	cfg.Servers = append(pve.ServerConfigurations{}, pve.ServerConfiguration{URL: host + "/api2/json"})
	cfg.DefaultHeader["Authorization"] = os.Getenv("PVE_TOKEN")

	client := pve.NewAPIClient(cfg)
	node := envOr("PVE_NODE", "pve1")
	vmid64, _ := strconv.ParseInt(envOr("PVE_VMID", "100"), 10, 32)
	vmid := int32(vmid64)

	target := pve.Target{Kind: pve.TargetKindQemu, Node: node, Vmid: vmid}
	opts := pve.RetryOptions{
		MaxRetries:        20,
		InitialDelay:      250 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := client.ConnectTerminalResilient(ctx, target, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	session.OnMessage = func(text string) { fmt.Print(text) }
	session.OnReconnect = func(attempt int) { fmt.Printf("\n[reconnected after %d attempts]\n", attempt) }
	session.OnGiveUp = func(err error) { fmt.Printf("\n[retries exhausted: %v]\n", err) }

	_ = session.Send("date\n")
	deadline := time.Now().Add(5 * time.Minute)
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			break
		case <-tick.C:
			_ = session.Send("date\n")
		}
	}
	_ = session.Close()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
