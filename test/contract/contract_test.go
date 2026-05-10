//go:build contract

// Package contract contains live-gateway smoke tests. Invoked via
// `make test-contract` with:
//
//	ZEAM_API_URL=https://api-gateway.zeam.app
//	ZEAM_CONTRACT_TESTS=1
//	ZEAM_CONTRACT_TOKEN=<bearer>
//
// Never run in default CI without explicit intent. These tests exercise
// the gateway directly and may trigger real OTP delivery if the
// ZEAM_CONTRACT_START_OTP env var is also set.
package contract

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ZeamMoney/zeam-sdk-go"
)

func TestHealthzHandshake(t *testing.T) {
	if os.Getenv("ZEAM_CONTRACT_TESTS") != "1" {
		t.Skip("set ZEAM_CONTRACT_TESTS=1 to run")
	}
	baseURL := os.Getenv("ZEAM_API_URL")
	if baseURL == "" {
		t.Skip("set ZEAM_API_URL")
	}

	client, err := zeam.New(
		zeam.WithEnvironment(zeam.EnvironmentCustom(baseURL)),
		zeam.WithInsecureTransport(),
		zeam.WithSkipVersionCheck(),
	)
	if err != nil {
		t.Fatalf("construct client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.Health().Get(ctx); err != nil {
		t.Fatalf("healthz: %v", err)
	}
}
