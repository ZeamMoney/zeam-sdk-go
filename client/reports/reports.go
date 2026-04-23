// Package reports wraps /v1/reports/*, /v1/market-prices, and
// /v1/network/status. These are read-only endpoints; retries are applied
// automatically by the transport layer (ADR 0008 R6).
package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Client wraps /v1/reports/*, /v1/market-prices, /v1/network/status.
type Client struct{ D client.Doer }

// New constructs a reports client.
func New(d client.Doer) *Client { return &Client{D: d} }

// APIUsage calls GET /v1/reports/api-usage?from=…&to=….
func (c *Client) APIUsage(ctx context.Context, sess *auth.Session, from, to string) (json.RawMessage, error) {
	q := url.Values{"from": {from}, "to": {to}}
	var out json.RawMessage
	err := client.Call(ctx, c.D, http.MethodGet, "/v1/reports/api-usage", q, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// WalletBalance calls GET /v1/reports/wallet-balance?wallet=….
func (c *Client) WalletBalance(ctx context.Context, sess *auth.Session, wallet string) (json.RawMessage, error) {
	q := url.Values{"wallet": {wallet}}
	var out json.RawMessage
	err := client.Call(ctx, c.D, http.MethodGet, "/v1/reports/wallet-balance", q, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// MarketPrice calls GET /v1/market-prices/{pair}?type=mid|buy-a|buy-b.
func (c *Client) MarketPrice(ctx context.Context, sess *auth.Session, pair, priceType string) (json.RawMessage, error) {
	q := url.Values{"type": {priceType}}
	var out json.RawMessage
	err := client.Call(ctx, c.D, http.MethodGet, fmt.Sprintf("/v1/market-prices/%s", pair), q, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// NetworkStatus calls GET /v1/network/status.
func (c *Client) NetworkStatus(ctx context.Context, sess *auth.Session) (json.RawMessage, error) {
	var out json.RawMessage
	err := client.Call(ctx, c.D, http.MethodGet, "/v1/network/status", nil, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}
