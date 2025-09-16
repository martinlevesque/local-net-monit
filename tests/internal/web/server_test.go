package web_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/martinlevesque/local-net-monit/internal/httpTooling"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"github.com/martinlevesque/local-net-monit/internal/web"
)

func TestMain(m *testing.M) {
	// setup
	netScanner := &networking.NetScanner{NotifyChangesToChannel: false}
	port1 := networking.Port{
		PortNumber: 80,
		Verified:   false,
		Notes:      "",
	}
	port2 := networking.Port{
		PortNumber: 81,
		Verified:   false,
		Notes:      "",
	}
	node := &networking.Node{
		IP:               "10.0.0.10",
		Name:             "node",
		LastPingDuration: time.Duration(1),
		Ports:            []networking.Port{port1, port2},
		Online:           true,
	}
	netScanner.NodeStatuses.Store("10.0.0.10", node)

	httpServer := web.BootstrapHttpServer(netScanner)

	// run tests
	code := m.Run()

	// teardown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if httpServer != nil {
		httpServer.Shutdown(ctx)
	}

	os.Exit(code)
}

func TestWebPrepareTemplates_Success(t *testing.T) {
	templates := web.PrepareTemplates()

	if len(templates) == 0 {
		t.Fatal("expected at least one template, but got none")
	}

	index_template := templates["index.html"]

	if index_template == nil {
		t.Fatal("expected an index.html template, but got nil")
	}

	t.Logf("Successfully prepared %d templates", len(templates))
}

func TestWebBootstrapHttpServer_Root_Success(t *testing.T) {
	status, body := httpTooling.Get("", "/")

	if status != "200 OK" {
		t.Fatalf("expected status 200 OK, but got: %s", status)
	}

	if !strings.Contains(body, "<title>") {
		t.Fatal("expected a non-empty body, but got none")
	}
}

func TestWebBootstrapHttpServer_VerifyAll_Success(t *testing.T) {
	os.Setenv("STATUS_PUBLIC_PORTS", "false")
	os.Setenv("STATUS_LOCAL_PORTS", "true")

	_, responseBodyStatus := httpTooling.Get(
		"",
		"/status",
	)

	if responseBodyStatus != "{\"status\":\"NOK\"}" {
		t.Fatalf("expected status NOK with status endpoint first, got %s", responseBodyStatus)
	}

	body := make(map[string]interface{})

	body["ip"] = "10.0.0.10"
	body["verified"] = true

	status, responseBody, _ := httpTooling.Post(
		"",
		"/ports/verify-all",
		body,
	)

	if status != "200 OK" {
		t.Fatalf("expected status 200 OK, but got: %s", status)
	}

	if responseBody != "{\"status\":\"success\"}" {
		t.Fatalf("expected success status response, but got: %s", responseBody)
	}

	_, responseBody = httpTooling.Get(
		"",
		"/status",
	)

	if responseBody != "{\"status\":\"OK\"}" {
		t.Fatalf("expected status OK with status endpoint %s", responseBody)
	}
}
