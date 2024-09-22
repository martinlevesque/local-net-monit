package networking_test

import (
	"github.com/martinlevesque/local-net-monit/internal/networking"
	"testing"
)

func TestNetworkingResolvePing_Success(t *testing.T) {
	host := "127.0.0.1"

	result, err := networking.ResolvePing(host)

	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}

	if result.Duration <= 0 {
		t.Fatalf("expected a valid duration, but got: %v", result.Duration)
	}

	t.Logf("Ping to %s successful with duration: %v", host, result.Duration)
}

func TestNetworkingResolvePing_InvalidIP(t *testing.T) {
	host := "invalid-ip"

	_, err := networking.ResolvePing(host)

	if err == nil {
		t.Fatal("expected an error due to invalid IP, but got nil")
	}

	t.Logf("Correctly failed ping to %s with error: %v", host, err)
}
