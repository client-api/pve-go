// Example: connect to a Proxmox host with a self-signed certificate.
//
// The PVE web UI ships with a self-signed cert by default. Production
// setups should use a real CA-signed cert (Let's Encrypt via the
// Proxmox UI), but home-lab and dev setups commonly need to opt out
// of cert verification.
//
// **Security note:** disabling verification is vulnerable to MITM.
// Use only on trusted networks, or pin a custom CertPool instead.
//
// Run with:
//
//	PVE_HOST=https://pve.example.com:8006 \
//	PVE_TOKEN='PVEAPIToken=root@pam!auto=...' \
//	PVE_NODE=orca PVE_VMID=100 \
//	go run ./examples/insecure_tls
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	pve "github.com//"
	gws "github.com/gorilla/websocket"
)

func main() {
	host := envOr("PVE_HOST", "https://localhost:8006")

	// ── 1. REST: install a Transport that skips cert verification.
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}

	cfg := pve.NewConfiguration()
	cfg.Servers = pve.ServerConfigurations{
		pve.ServerConfiguration{URL: host + "/api2/json"},
	}
	cfg.DefaultHeader["Authorization"] = os.Getenv("PVE_TOKEN")
	cfg.HTTPClient = &http.Client{Transport: insecureTransport}

	// ── 2. WebSocket: replace the package-level DefaultTransport with
	//    a GorillaTransport whose Dialer carries the same insecure
	//    TLSClientConfig. This is resolved at connect-time, so it
	//    must be set before the first ConnectTerminal / ConnectVnc
	//    call (here, before NewAPIClient is even fine).
	pve.DefaultTransport = pve.GorillaTransport{
		Dialer: &gws.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
	}

	client := pve.NewAPIClient(cfg)

	// /version is a trivial REST sanity check (no model decoding —
	// avoids the int32 `mem` field overflow on big-RAM hosts).
	if _, resp, err := client.VersionAPI.VersionVersion(context.Background()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "REST sanity check:", err)
		os.Exit(1)
	} else {
		fmt.Printf("Connected (insecure TLS): /version → %d\n", resp.StatusCode)
	}

	if os.Getenv("PVE_NODE") == "" || os.Getenv("PVE_VMID") == "" {
		fmt.Println("(skip terminal: set PVE_NODE and PVE_VMID to test the WebSocket leg)")
		return
	}
	vmid64, _ := strconv.ParseInt(os.Getenv("PVE_VMID"), 10, 32)
	target := pve.Target{Kind: pve.TargetKindQemu, Node: os.Getenv("PVE_NODE"), Vmid: int32(vmid64)}

	session, err := client.ConnectTerminal(context.Background(), target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect terminal:", err)
		os.Exit(1)
	}
	session.OnMessage = func(text string) { fmt.Print(text) }
	_ = session.Send("uname -a\n")
	time.Sleep(3 * time.Second)
	_ = session.Close()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
