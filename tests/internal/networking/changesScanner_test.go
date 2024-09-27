package networking_test

import (
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"testing"
)

func TestVerifyPort_Success(t *testing.T) {
	portHttp := networking.Port{
		PortNumber: 80,
		Verified:   false,
		Notes:      "HTTP",
	}

	otherPort := networking.Port{
		PortNumber: 8000,
		Verified:   false,
		Notes:      "hello",
	}

	node := &networking.Node{
		IP:    "127.0.0.1",
		Ports: []networking.Port{portHttp, otherPort},
	}

	result := node.VerifyPort(8000, true, "hello2")

	if !result {
		t.Fatalf("expected true, but got false")
	}

	if !node.Ports[1].Verified {
		t.Fatalf("expected port to be verified, but it's not")
	}

	if node.Ports[1].Notes != "hello2" {
		t.Fatalf("expected notes to be 'hello2', but got '%s'", node.Ports[1].Notes)
	}
}
