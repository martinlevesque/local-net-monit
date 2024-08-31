package web_test

import (
	"context"
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"github.com/martinlevesque/local-net-monit/internal/web"
	"strings"
	"testing"
	"time"
)

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

func TestWebBootstrapHttpServer_Success(t *testing.T) {
	netScanner := &networking.NetScanner{}
	httpServer := web.BootstrapHttpServer(netScanner)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, body := web.Get("", "/")

	if status != "200 OK" {
		t.Fatalf("expected status 200 OK, but got: %s", status)
	}

	if !strings.Contains(body, "<title>") {
		t.Fatal("expected a non-empty body, but got none")
	}

	httpServer.Shutdown(ctx)

}
