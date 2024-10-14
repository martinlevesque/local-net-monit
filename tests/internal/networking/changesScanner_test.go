package networking_test

import (
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"testing"
	"time"
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

func TestVerifyNodeUptimeTimeouts_HappyPath(t *testing.T) {
	ns := networking.NetScanner{}

	// Create two nodes: one that has been online recently, and one that timed out
	recentTime := time.Now().Add(-2 * time.Minute)  // Within the timeout window
	timeoutTime := time.Now().Add(-200 * time.Hour) // Exceeds the timeout window

	// Add nodes to the NodeStatuses map
	ns.NodeStatuses.Store("node1", &networking.Node{LastOnlineAt: &recentTime})
	ns.NodeStatuses.Store("node2", &networking.Node{LastOnlineAt: &timeoutTime})

	ns.VerifyNodeUptimeTimeouts()

	// Check that node1 is still in the map
	_, ok1 := ns.NodeStatuses.Load("node1")

	if !ok1 {
		t.Errorf("node1 should not have been deleted")
	}

	// Check that node2 has been removed from the map
	_, ok2 := ns.NodeStatuses.Load("node2")

	if ok2 {
		t.Errorf("node2 should have been deleted due to timeout")
	}
}
