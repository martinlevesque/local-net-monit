package networking_test

import (
	"testing"
	"time"

	"github.com/martinlevesque/local-net-monit/internal/networking"
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
func TestVerifyIp_Success(t *testing.T) {
	node := &networking.Node{
		IP:   "127.0.0.1",
		Name: "origname",
	}

	node.VerifyIp("newname", true)

	if node.Name != "newname" {
		t.Fatalf("expected to change name")
	}
}

func TestVerifyNodeUptimeTimeouts_HappyPath(t *testing.T) {
	ns := networking.NetScanner{}

	// Create two nodes: one that has been online recently, and one that timed out
	recentTime := time.Now().Add(-2 * time.Minute)  // Within the timeout window
	timeoutTime := time.Now().Add(-200 * time.Hour) // Exceeds the timeout window

	// Convert times to RFC3339 strings as expected by the Node struct
	recentTimeStr := recentTime.Format(time.RFC3339)
	timeoutTimeStr := timeoutTime.Format(time.RFC3339)

	// Add nodes to the NodeStatuses map
	ns.NodeStatuses.Store("node1", &networking.Node{LastOnlineAt: recentTimeStr})
	ns.NodeStatuses.Store("node2", &networking.Node{LastOnlineAt: timeoutTimeStr})

	ns.VerifyNodeUptimeTimeouts()

	// Check that node1 is still in the map
	_, node1Ok := ns.NodeStatuses.Load("node1")

	if !node1Ok {
		t.Errorf("node1 should not have been deleted")
	}

	// Check that node2 has been removed from the map
	_, node2Ok := ns.NodeStatuses.Load("node2")

	if node2Ok {
		t.Errorf("node2 should have been deleted due to timeout")
	}
}
