// Package health wraps GET /healthz.
package health

import (
	"context"
	"net/http"

	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Response mirrors the gateway /healthz body per ADR 0003 Finding #4
// (once ratified: `{ok, status, uptime_ms, version}`).
type Response struct {
	OK       bool   `json:"ok"`
	Status   string `json:"status"`
	UptimeMS int64  `json:"uptime_ms"`
	Version  string `json:"version"`
}

// Client wraps the /healthz endpoint.
type Client struct{ D client.Doer }

// New constructs a health client.
func New(d client.Doer) *Client { return &Client{D: d} }

// Get calls GET /healthz. The endpoint is unauthenticated and returns
// bare JSON (not SPEC §18).
func (c *Client) Get(ctx context.Context) (*Response, error) {
	var out Response
	err := client.Call(ctx, c.D, http.MethodGet, "/healthz", nil, nil, 0, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
