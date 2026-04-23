// Package payments wraps /v1/payments, /v1/quotes, and /v1/orders.
//
// All three endpoints are authenticated by a Business/Firebase session.
// The DTOs below expose the subset of fields the recipes need; each
// method returns raw JSON alongside typed fields so partners can access
// upstream-specific extensions without waiting for an SDK bump.
package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ZeamMoney/zeam-sdk-go/auth"
	"github.com/ZeamMoney/zeam-sdk-go/client"
)

// Client wraps /v1/payments, /v1/quotes, /v1/orders.
type Client struct{ D client.Doer }

// New constructs a payments client.
func New(d client.Doer) *Client { return &Client{D: d} }

// Payment is the minimal payment record. Full schema lives in the
// gateway-owned OpenAPI spec; Raw carries the upstream payload verbatim.
type Payment struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	Amount    string          `json:"amount"`
	CreatedAt string          `json:"created_at"`
	Raw       json.RawMessage `json:"-"`
}

// CreatePayment calls POST /v1/payments.
func (c *Client) CreatePayment(ctx context.Context, sess *auth.Session, body any) (*Payment, error) {
	var out Payment
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/payments", nil, sess, auth.TrackBusiness, "", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPayments calls GET /v1/payments with the supplied query parameters.
func (c *Client) ListPayments(ctx context.Context, sess *auth.Session, q url.Values) ([]Payment, error) {
	var out []Payment
	err := client.Call(ctx, c.D, http.MethodGet, "/v1/payments", q, sess, auth.TrackBusiness, "", nil, &out)
	return out, err
}

// GetPayment calls GET /v1/payments/{id}.
func (c *Client) GetPayment(ctx context.Context, sess *auth.Session, id string) (*Payment, error) {
	var out Payment
	err := client.Call(ctx, c.D, http.MethodGet, fmt.Sprintf("/v1/payments/%s", id), nil, sess, auth.TrackBusiness, "", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Quote is the /v1/quotes record.
type Quote struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Raw    json.RawMessage `json:"-"`
}

// CreateQuote calls POST /v1/quotes.
func (c *Client) CreateQuote(ctx context.Context, sess *auth.Session, body any) (*Quote, error) {
	var out Quote
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/quotes", nil, sess, auth.TrackBusiness, "", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Order is the /v1/orders record.
type Order struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Raw    json.RawMessage `json:"-"`
}

// CreateOrder calls POST /v1/orders.
func (c *Client) CreateOrder(ctx context.Context, sess *auth.Session, body any) (*Order, error) {
	var out Order
	err := client.Call(ctx, c.D, http.MethodPost, "/v1/orders", nil, sess, auth.TrackBusiness, "", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrder calls POST /v1/orders/{id}/cancel.
func (c *Client) CancelOrder(ctx context.Context, sess *auth.Session, id string) error {
	return client.Call(ctx, c.D, http.MethodPost, fmt.Sprintf("/v1/orders/%s/cancel", id), nil, sess, auth.TrackBusiness, "", nil, nil)
}
