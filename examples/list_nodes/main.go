// Example: list cluster nodes.
//
// Run with:
//
//	PVE_HOST=https://pve.example.com:8006 \
//	PVE_TOKEN='PVEAPIToken=root@pam!auto=...' \
//	go run ./examples/list_nodes
package main

import (
	"context"
	"fmt"
	"os"

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
	resp, _, err := client.NodesAPI.NodesGetNodes(context.Background()).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "list nodes:", err)
		os.Exit(1)
	}
	nodes := resp.GetData()
	fmt.Printf("Found %d node(s):\n", len(nodes))
	for _, n := range nodes {
		fmt.Printf("  - %v (status=%v, cpu=%v, mem=%v/%v)\n",
			n.Node, n.Status, n.Cpu, n.Mem, n.Maxmem)
	}
}
